// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

// Package cmd implements the blackstork-cli commands.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/blackstork-io/blackstork-cli/engine"
	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
)

var (
	tracer      trace.Tracer
	rootSpan    trace.Span
	rootCleanup func(context.Context) error
	rootCtx     context.Context
	cliArgs     = struct {
		sourceDir string
		noColor   bool
		debug     bool
	}{}
	rawArgs = struct {
		sourceDir string
		logOutput string
		logLevel  string
		verbose   bool
		noColor   bool
		debug     bool
		inputs    []string
	}{}
	debugDir = ".blackstork/debug"
	env      = struct {
		otelpEnabled bool
		otelpURL     string
	}{
		otelpEnabled: false,
		otelpURL:     "https://otelp.blackstork.io",
	}
)

func init() {
	rootCmd.PersistentFlags().
		StringVar(&rawArgs.sourceDir, "source-dir", ".", "a path to a directory with *.blackstork.hcl files")
	rootCmd.PersistentFlags().StringVar(&rawArgs.logOutput, "log-format", "plain", "format of the logs (plain or json)")
	rootCmd.PersistentFlags().StringVar(
		&rawArgs.logLevel, "log-level", "warn",
		fmt.Sprintf("logging level (%s)", utils.GetLogLevelsString()),
	)
	rootCmd.PersistentFlags().BoolVarP(&rawArgs.verbose, "verbose", "v", false, "a shortcut to --log-level debug")
	rootCmd.PersistentFlags().BoolVar(&rawArgs.debug, "debug", false, "enables debug mode")
	rootCmd.PersistentFlags().
		StringArrayVarP(&rawArgs.inputs, "input", "i", []string{}, "template inputs in the format of <name>=<value>. The flag can be repeated")

	if otelpURL := os.Getenv("BLACKSTORK_OTELP_URL"); otelpURL != "" {
		env.otelpURL = otelpURL
	}
	if os.Getenv("BLACKSTORK_OTELP_ENABLED") == "true" {
		env.otelpEnabled = true
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "blackstork-cli",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		if rawArgs.debug {
			rootCleanup, err = utils.SetupStdout(ctx, debugDir, version)
		} else if env.otelpEnabled {
			rootCleanup, err = utils.SetupOtelp(ctx, env.otelpURL, version)
		}
		if err != nil {
			return err
		}
		defer func() {
			rootCtx = ctx
		}()
		tracer = otel.Tracer("blackstork-cli/cmd")
		ctx, rootSpan = tracer.Start(ctx, "Command", trace.WithAttributes(
			attribute.String("command", cmd.Name()),
		))

		err = validateDir(rawArgs.sourceDir)
		if err != nil {
			return err
		}
		cliArgs.sourceDir = rawArgs.sourceDir

		var levelName string
		if rawArgs.verbose || rawArgs.debug {
			levelName = "debug"
		} else {
			levelName = rawArgs.logLevel
		}

		logFormat := strings.ToLower(strings.TrimSpace(rawArgs.logOutput))

		err = utils.ConfigureLogging(version, levelName, logFormat, env.otelpEnabled)
		if err != nil {
			return err
		}

		inputKVs := utils.FnMap(
			slices.DeleteFunc(
				utils.FnMap(
					rawArgs.inputs,
					func(val string) []string {
						return strings.SplitN(val, "=", 2)
					},
				),
				func(pair []string) bool {
					return len(pair) != 2
				},
			),
			func(pair []string) *appctx.InputKeyValue {
				return &appctx.InputKeyValue{
					Key:   pair[0],
					Value: pair[1],
				}
			},
		)

		ctx = appctx.WithInputs(ctx, inputKVs)

		slog.DebugContext(ctx, "Starting the execution")
		if strings.Contains(version, "-dev") {
			slog.WarnContext(ctx, "This is a dev version of the software!", "version", version)
		}
		cmd.SetContext(ctx)
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	ctx := appctx.NewCLI()
	exitCode := 0

	ctx = appctx.WithTracer(ctx, tracer)

	err := executeWithRecover(ctx, rootCmd)
	if err != nil {
		exitCode = 1
	}
	if rootSpan != nil {
		if err != nil {
			rootSpan.RecordError(err)
			rootSpan.SetStatus(codes.Error, err.Error())
		} else {
			rootSpan.SetStatus(codes.Ok, "success")
		}
		rootSpan.End()
	}
	if rootCleanup != nil {
		if cleanupErr := rootCleanup(rootCtx); cleanupErr != nil {
			slog.ErrorContext(rootCtx, "Failed to clean up telemetry", "err", cleanupErr)
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}

func executeWithRecover(ctx context.Context, cmd *cobra.Command) (err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(rootCtx, "Panic error caught", "error", r, "stack", string(debug.Stack()))
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return cmd.ExecuteContext(ctx)
}

func exitCommand(eng *engine.Engine, cmd *cobra.Command, diags diagnostics.Diag) (err error) {
	err = eng.Cleanup()
	if diags.HasErrors() {
		err = diags
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
	} else if err != nil {
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
	} else {
		err = nil
	}

	eng.PrintDiagnostics(os.Stderr, diags, !cliArgs.noColor)
	return err
}

func validateDir(dir string) error {
	info, err := os.Stat(dir)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("`%s` doesn't exist", dir)
	case errors.Is(err, os.ErrPermission):
		return fmt.Errorf("can't access `%s`", dir)
	default:
		return fmt.Errorf("error validating `%s`: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("`%s` is not a directory", dir)
	}
	return nil
}
