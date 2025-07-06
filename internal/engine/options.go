package engine

import (
	"io"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"

	"github.com/blackstork-io/fabric/plugins/builtin"
	"github.com/blackstork-io/fabric/internal/plugin"
)

const (
	defaultRegistryBaseURL = "https://registry.blackstork.io"
	defaultCacheDir        = ".fabric"
	defaultLockFile        = ".fabric-lock.json"
)

// Options is a set of options for the engine.
type Options struct {
	registryBaseURL string
	cacheDir        string
	builtin         *plugin.Schema
	log             *slog.Logger
	tracer          trace.Tracer
}

var defaultLogger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
	Level: slog.LevelError,
}))

var defaultOptions = Options{
	registryBaseURL: defaultRegistryBaseURL,
	cacheDir:        defaultCacheDir,
	log:             defaultLogger,
	tracer:          nooptrace.Tracer{},
	builtin:         builtin.Plugin("v0.0.0", defaultLogger, nil),
}

type Option func(*Options)

// WithRegistryBaseURL sets the registry base URL. Default is "https://registry.blackstork.io".
func WithRegistryBaseURL(url string) Option {
	return func(o *Options) {
		o.registryBaseURL = url
	}
}

// WithCacheDir sets the cache directory. Default is ".fabric".
func WithCacheDir(dir string) Option {
	return func(o *Options) {
		o.cacheDir = dir
	}
}

// WithBuiltIn sets the built-in plugin.
func WithBuiltIn(builtin *plugin.Schema) Option {
	return func(o *Options) {
		o.builtin = builtin
	}
}

// WithLogger sets the logger. Default is a logger that discards all logs.
func WithLogger(log *slog.Logger) Option {
	return func(o *Options) {
		o.log = log
	}
}

// WithTracer sets the tracer. Default is noop tracer.
func WithTracer(tracer trace.Tracer) Option {
	return func(o *Options) {
		o.tracer = tracer
	}
}
