package fabctx

import (
	"context"
	"log/slog"
)

type logKeyT struct{}

var logKey = logKeyT{}

func GetLog(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if ec, ok := ctx.Value(logKey).(*slog.Logger); ok {
			return ec
		}
	}
	return slog.Default()
}

func WithLog(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, logKey, log)
}
