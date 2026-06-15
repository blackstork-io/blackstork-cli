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
)

type inputsKeyT struct{}

var inputsKey = inputsKeyT{}

type InputKeyValue struct {
	Key   string
	Value string
}

func Inputs(ctx context.Context) []*InputKeyValue {
	if ctx != nil {
		if ec, ok := ctx.Value(inputsKey).([]*InputKeyValue); ok {
			return ec
		}
	}
	return []*InputKeyValue{}
}

func WithInputs(ctx context.Context, inputs []*InputKeyValue) context.Context {
	return context.WithValue(ctx, inputsKey, inputs)
}
