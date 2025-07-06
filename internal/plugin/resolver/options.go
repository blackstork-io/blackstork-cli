package resolver

import (
	"io"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// options for the resolver.
type options struct {
	log     *slog.Logger
	tracer  trace.Tracer
	sources []Source
}

var defaultOptions = options{
	log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	sources: []Source{},
	tracer:  tracenoop.Tracer{},
}

// Option is a functional option for the resolver.
type Option func(*options)

// WithLogger sets the log for the resolver.
func WithLogger(log *slog.Logger) Option {
	return func(o *options) {
		o.log = log
	}
}

// WithSources sets the sources for the resolver.
func WithSources(sources ...Source) Option {
	return func(o *options) {
		o.sources = sources
	}
}

// WithTracer sets the tracer for the resolver.
func WithTracer(tracer trace.Tracer) Option {
	return func(o *options) {
		o.tracer = tracer
	}
}
