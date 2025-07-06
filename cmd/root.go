package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/blackstork-io/fabric/internal/fabctx"
	"github.com/blackstork-io/fabric/internal/engine"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/utils"
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
	}{}
	debugDir = ".fabric/debug"
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
		StringVar(&rawArgs.sourceDir, "source-dir", ".", "a path to a directory with *.fabric files")
	rootCmd.PersistentFlags().StringVar(&rawArgs.logOutput, "log-format", "plain", "format of the logs (plain or json)")
	rootCmd.PersistentFlags().StringVar(
		&rawArgs.logLevel, "log-level", "info",
		fmt.Sprintf("logging level (%s)", utils.GetLogLevelsString()),
	)
	rootCmd.PersistentFlags().BoolVarP(&rawArgs.verbose, "verbose", "v", false, "a shortcut to --log-level debug")
	rootCmd.PersistentFlags().BoolVar(&rawArgs.debug, "debug", false, "enables debug mode")

	if otelpURL := os.Getenv("FABRIC_OTELP_URL"); otelpURL != "" {
		env.otelpURL = otelpURL
	}
	if os.Getenv("FABRIC_OTELP_ENABLED") == "true" {
		env.otelpEnabled = true
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "fabric",
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
		tracer = otel.Tracer("fabric/cmd")
		ctx, rootSpan = tracer.Start(ctx, "Command", trace.WithAttributes(
			attribute.String("command", cmd.Name()),
		))

		err = validateDir(rawArgs.sourceDir)
		if err != nil {
			return
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
	var ctx context.Context = fabctx.New()
	exitCode := 0

	ctx = fabctx.WithTracer(ctx, tracer)

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
		rootCleanup(rootCtx)
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
	diags.Extend(eng.Cleanup())
	if diags.HasErrors() {
		err = diags
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
