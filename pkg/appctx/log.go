// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package appctx

import (
	"context"
	"log/slog"
)

type logKeyT struct{}

var logKey = logKeyT{}

func Log(ctx context.Context) *slog.Logger {
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
