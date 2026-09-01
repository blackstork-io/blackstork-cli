// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

// Package multilog combines multiple slog handlers.
package multilog

import (
	"context"
	"errors"
	"log/slog"
)

// Handler is a log handler that forwards log records to multiple handlers.
// It is useful when you want to log to multiple destinations, e.g. console and opentelemetry.
type Handler struct {
	Level    slog.Level
	Handlers []slog.Handler
}

// Enabled returns true if the log level is enabled.
func (multi Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return multi.Level <= level
}

// Handle forwards the log record to all handlers.
func (multi Handler) Handle(ctx context.Context, record slog.Record) error {
	var errs []error
	for _, handler := range multi.Handlers {
		err := handler.Handle(ctx, record)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// WithAttrs returns a new Handler with the given attributes.
func (multi Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(multi.Handlers))
	for i, handler := range multi.Handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	multi.Handlers = handlers
	return multi
}

// WithGroup returns a new Handler with the given group name.
func (multi Handler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(multi.Handlers))
	for i, handler := range multi.Handlers {
		handlers[i] = handler.WithGroup(name)
	}
	multi.Handlers = handlers
	return multi
}
