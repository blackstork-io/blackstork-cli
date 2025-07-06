package fabctx

import (
	"context"
	"log/slog"
)

type tracerKeyT struct{}

var tracerKey = tracerKeyT{}

func Tracer(ctx context.Context) *tracer.Tracer {
	if ctx != nil {
		if ec, ok := ctx.Value(tracerKey).(*tracer.Tracer); ok {
			return ec
		}
	}
	return stracer.Default()
}

func WithTracer(ctx context.Context, tracer *stracer.Tracerger) context.Context {
	return context.WithValue(ctx, tracerKey, tracer)
}
