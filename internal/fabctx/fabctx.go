package fabctx

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/hashicorp/hcl/v2"
)

// FabCtx is a context that can be used to cancel the main context.
// It is used to handle graceful shutdowns for the CLI execution flow.
type FabCtx struct {
	mainCtx    context.Context

	// Template evaluation context
	evalCtx    *hcl.EvalContext
}

var _ context.Context = (*FabCtx)(nil)

func (ctx *FabCtx) Deadline() (deadline time.Time, ok bool) {
	return ctx.mainCtx.Deadline()
}

func (ctx *FabCtx) Done() <-chan struct{} {
	return ctx.mainCtx.Done()
}

func (ctx *FabCtx) Err() error {
	return ctx.mainCtx.Err()
}

type getFabCtxT struct{}

var getFabCtx = getFabCtxT{}

func Get(ctx context.Context) *FabCtx {
	if ctx == nil {
		return nil
	}
	if fc, ok := ctx.(*FabCtx); ok {
		return fc
	}
	fc := ctx.Value(getFabCtx)
	if fc == nil {
		return nil
	}
	if fc, ok := fc.(*FabCtx); ok {
		return fc
	}
	return nil
}

func (ctx *FabCtx) Value(v any) any {
	switch v.(type) {
	case getFabCtxT:
		return ctx
	case evalCtxKeyT:
		return ctx.evalCtx
	default:
		// fabCtx is the root context
		return nil
	}
}

type fabCtxOpts struct {
	signals bool
}

type Option func(*fabCtxOpts)

func NoSignals(opts *fabCtxOpts) {
	opts.signals = false
}

// Returns a cli-appropriate context (cancelable by ctrl+c).
func New(options ...Option) *FabCtx {
	opts := fabCtxOpts{
		signals: true,
	}
	for _, opt := range options {
		opt(&opts)
	}

	ctx := FabCtx{
		evalCtx: newEvalContext(),
	}

	if !opts.signals {
		ctx.mainCtx = context.Background()
		return &ctx
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
// 			case 1:
// 				slog.ErrorContext(&ctx, "Received second os.Interrupt")
// 				cleanupCancel(fmt.Errorf("got termination request (forceful)"))
			default:
				slog.ErrorContext(&ctx, "Rough exit with multiple interrupts received")
				panic("Interruped")
			}
			caught++
		}
	}()
	return &ctx
}
