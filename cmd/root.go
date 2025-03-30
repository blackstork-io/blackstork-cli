package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/golang-cz/devslog"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-colorable"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/term"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/cmd/internal/multilog"
	"github.com/blackstork-io/fabric/cmd/internal/telemetry"
	"github.com/blackstork-io/fabric/engine"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/utils/slogutil"
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
	rootCmd.PersistentFlags().StringVar(&rawArgs.sourceDir, "source-dir", ".", "a path to a directory with *.fabric files")
	rootCmd.PersistentFlags().StringVar(&rawArgs.logOutput, "log-format", "plain", "format of the logs (plain or json)")
	rootCmd.PersistentFlags().StringVar(
		&rawArgs.logLevel, "log-level", "info",
		fmt.Sprintf("logging level (%s)", validLogLevels.String()),
	)
	rootCmd.PersistentFlags().BoolVar(&rawArgs.noColor, "no-color", false, "disable colorization of the output (logs and diagnostics), if supported by the terminal and the log format")
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
			rootCleanup, err = telemetry.SetupStdout(ctx, debugDir, version)
		} else if env.otelpEnabled {
			rootCleanup, err = telemetry.SetupOtelp(ctx, env.otelpURL, version)
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
		err = validateDir("source dir", rawArgs.sourceDir)
		if err != nil {
			return
		}
		cliArgs.sourceDir = rawArgs.sourceDir
		cliArgs.noColor = rawArgs.noColor && term.IsTerminal(int(os.Stderr.Fd()))
		var level slog.Level
		if rawArgs.verbose || rawArgs.debug {
			level = slog.LevelDebug
		} else {
			level, err = validLogLevels.Find(rawArgs.logLevel)
			if err != nil {
				return
			}
		}
		opts := &slog.HandlerOptions{
			Level: level,
			// add source if in debug mode
			AddSource: level == slog.LevelDebug,
		}
		var handler slog.Handler
		switch strings.ToLower(strings.TrimSpace(rawArgs.logOutput)) {
		case "plain":
			if !cliArgs.noColor && level <= slog.LevelDebug {
				handler = devslog.NewHandler(os.Stderr, &devslog.Options{
					HandlerOptions: opts,
				})
			} else {
				var output io.Writer
				if cliArgs.noColor {
					output = os.Stderr
				} else {
					// only affects windows, noop on *nix
					output = colorable.NewColorable(os.Stderr)
				}

				handler = tint.NewHandler(
					output,
					&tint.Options{
						AddSource:   opts.AddSource,
						Level:       opts.Level,
						ReplaceAttr: opts.ReplaceAttr,
						NoColor:     cliArgs.noColor,
						TimeFormat:  time.DateTime,
					},
				)
			}
		case "json":
			handler = slog.NewJSONHandler(os.Stderr, opts)
		default:
			return fmt.Errorf("unknown log output '%s'", rawArgs.logOutput)
		}
		var logger *slog.Logger
		if env.otelpEnabled || rawArgs.debug {
			handler = multilog.Handler{
				Level: level,
				Handlers: []slog.Handler{
					handler,
					otelslog.NewHandler(
						"github.com/blackstork-io/fabric",
						otelslog.WithVersion(version),
					),
				},
			}
		}
		handler = slogutil.NewSourceRewriter(handler)
		logger = slog.New(handler)
		logger = logger.With("command", cmd.Name())
		slog.SetDefault(logger)
		slog.SetLogLoggerLevel(slog.LevelDebug)
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
	err := recoverExecute(ctx, rootCmd)
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
			slog.ErrorContext(rootCtx, "Failed to clean up", "error", cleanupErr)
		}
	}
	os.Exit(exitCode)
}

func recoverExecute(ctx context.Context, cmd *cobra.Command) (err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(rootCtx, "Panic", "error", r, "stack", string(debug.Stack()))
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
