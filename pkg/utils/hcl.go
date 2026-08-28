// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package utils

import (
	"errors"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
)

const (
	maxRefDepth = 10
)

func RangeStart(rng hcl.Range) hcl.Range {
	rng.Start = rng.End
	return rng
}

func RangeEnd(rng hcl.Range) hcl.Range {
	rng.End = rng.Start
	return rng
}

func ToHclsyntaxBody(body hcl.Body) *hclsyntax.Body {
	hclsyntaxBody, ok := body.(*hclsyntax.Body)
	if !ok {
		// Should never happen: hcl.Body for hcl documents is always *hclsyntax.Body
		panic("hcl.Body to *hclsyntax.Body failed")
	}
	return hclsyntaxBody
}

func EvalContextByVar(ctx *hcl.EvalContext, name string) *hcl.EvalContext {
	for ; ctx != nil; ctx = ctx.Parent() {
		if ctx.Variables == nil {
			continue
		}
		_, found := ctx.Variables[name]
		if found {
			return ctx
		}
	}
	return nil
}

func EvalContextByFunc(ctx *hcl.EvalContext, name string) *hcl.EvalContext {
	for ; ctx != nil; ctx = ctx.Parent() {
		if ctx.Functions == nil {
			continue
		}
		_, found := ctx.Functions[name]
		if found {
			return ctx
		}
	}
	return nil
}

type onceVal[V any] struct {
	state atomic.Int32
	fn    func() (V, diagnostics.Diag)
	res   V
	mu    sync.Mutex
}

func (o *onceVal[V]) do() (res V, diags diagnostics.Diag) {
	state := o.state.Load()
	switch {
	case state > 0:
		return o.res, nil
	case state < 0:
		diags = diagnostics.Diag{diagnostics.RepeatedError}
		return res, diags
	}
	o.mu.Lock()
	defer func() {
		o.fn = nil
		if state == 0 {
			// this is a panic
			o.state.Store(-1)
		}
		o.mu.Unlock()
	}()
	state = o.state.Load()
	switch {
	case state > 0:
		res = o.res
	case state < 0:
		diags = diagnostics.Diag{diagnostics.RepeatedError}
	default:
		res, diags = o.fn()
		if diags.HasErrors() {
			state = -1
		} else {
			o.res = res
			state = 1
		}
		o.state.Store(state)
	}
	return res, diags
}

// OnceVal returns a function that calls fn only once and caches the result.
// If fn returns diagnostics with errors, the function will return it only once,
// on subsequent calls it will return RepeatedError.
func OnceVal[V any](fn func() (V, diagnostics.Diag)) func() (V, diagnostics.Diag) {
	return (&onceVal[V]{fn: fn}).do
}

// Using pointers as unique ids, not accessing the data.
type RefId unsafe.Pointer

type RefHistory struct {
	mu   sync.Mutex
	refs []RefId
}

func NewRefHistory() *RefHistory {
	return &RefHistory{
		refs: make([]RefId, 0),
	}
}

func (hist *RefHistory) Add(refRange *hcl.Range) {
	hist.mu.Lock()
	hist.refs = append(hist.refs, RefId(refRange))
	hist.mu.Unlock()
}

func (hist *RefHistory) Pop() {
	hist.mu.Lock()
	if len(hist.refs) > 0 {
		hist.refs = hist.refs[:len(hist.refs)-1]
	}
	hist.mu.Unlock()
}

func (hist *RefHistory) IsRefAllowed() bool {
	return len(hist.refs) <= maxRefDepth
}

func (hist *RefHistory) Size() int {
	if hist == nil {
		return 0
	}
	return len(hist.refs)
}

func ParseStringIntoTraversalExpression(input string) (hclsyntax.Expression, error) {
	expr, diags := hclsyntax.ParseExpression([]byte(input), "<inline>", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, errors.New(diags.Error())
	}
	return expr, nil
}
