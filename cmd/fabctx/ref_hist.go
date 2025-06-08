package fabctx

import (
	"context"

	"github.com/blackstork-io/fabric/pkg/utils"
)

type refHistKeyT struct{}

var refHistKey = refHistKeyT{}

func GetRefHistory(ctx context.Context) *utils.RefHistory {
	if ctx != nil {
		if ec, ok := ctx.Value(refHistKey).(*utils.RefHistory); ok {
			return ec
		}
	}
	return utils.NewRefHistory()
}

func WithRefHistory(ctx context.Context, refHist *utils.RefHistory) context.Context {
	return context.WithValue(ctx, refHistKey, refHist)
}
