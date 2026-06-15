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
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/hashicorp/hcl/v2"
)

// AppCtx is a context that can be used to cancel the main context.
// It is used to handle graceful shutdowns for the CLI execution flow.
type AppCtx struct {
	mainCtx context.Context

	// Template evaluation context
	evalCtx *hcl.EvalContext
}

var _ context.Context = (*AppCtx)(nil)

func (ctx *AppCtx) Deadline() (deadline time.Time, ok bool) {
	return ctx.mainCtx.Deadline()
}

func (ctx *AppCtx) Done() <-chan struct{} {
	return ctx.mainCtx.Done()
}

func (ctx *AppCtx) Err() error {
	return ctx.mainCtx.Err()
}

type getAppCtxT struct{}

var getAppCtx = getAppCtxT{}

func Get(ctx context.Context) *AppCtx {
	if ctx == nil {
		return nil
	}
	if fc, ok := ctx.(*AppCtx); ok {
		return fc
	}
	fc := ctx.Value(getAppCtx)
	if fc == nil {
		return nil
	}
	if fc, ok := fc.(*AppCtx); ok {
		return fc
	}
	return nil
}

func (ctx *AppCtx) Value(v any) any {
	switch v.(type) {
	case getAppCtxT:
		return ctx
	case evalCtxKeyT:
		return ctx.evalCtx
	default:
		// appCtx is the root context
		return nil
	}
}

// New returns a simple generic context
func New() context.Context {
	ctx := AppCtx{
		evalCtx: newEvalContext(),
	}
	ctx.mainCtx = context.Background()
	return &ctx
}

// NewCLI returns a cli-appropriate context (cancelable by ctrl+c).
func NewCLI() context.Context {
	ctx := AppCtx{
		evalCtx: newCLIEvalContext(),
	}

	var mainCancel context.CancelCauseFunc
	ctx.mainCtx, mainCancel = context.WithCancelCause(context.Background())

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt)

	go func() {
		caught := 0
		for range c {
			switch caught {
			case 0:
				slog.WarnContext(&ctx, "Received os.Interrupt")
				mainCancel(fmt.Errorf("got termination request (gentle)"))
				// 	case 1:
				// 		slog.ErrorContext(&ctx, "Received second os.Interrupt")
				// 		cleanupCancel(fmt.Errorf("got termination request (forceful)"))
			default:
				slog.ErrorContext(&ctx, "Rough exit with multiple interrupts received")
				panic("Interruped")
			}
			caught++
		}
	}()
	return &ctx
}
