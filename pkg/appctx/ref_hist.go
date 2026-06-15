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

	"github.com/blackstork-io/blackstork-cli/pkg/utils"
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
