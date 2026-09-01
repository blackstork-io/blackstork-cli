// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

// Package plugintest provides helpers for testing BlackStork plugins.
package plugintest

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

var uniqueID atomic.Uint32

// Generates a unique filename for a test file
// This allows to merge file maps from config and args
func getFileName() string {
	return fmt.Sprintf("generated-test-file-%d.fabric", uniqueID.Add(1))
}

// ReencodeCTY works around tests that break as soon as we add
// schemas with default values. This function is a workaround.
// It reencodes provided cty.Value to hcl text and then re-parses that text
// in accordance to spec. Ugly hack, but there's over a 100 tests in need of
// a rewrite that can't be automated with regex or similar.
//
// Deprecated: use plugintest.NewTestDecoder
func ReencodeCTY(t *testing.T, spec *dataspec.RootSpec, val cty.Value, asserts [][]diagtest.Assert) *dataspec.Block {
	t.Helper()
	ty := val.Type()
	if !ty.IsMapType() && !ty.IsObjectType() {
		panic("Can't handle type " + ty.FriendlyName())
	}
	f := hclwrite.NewEmptyFile()
	b := f.Body()
	for it := val.ElementIterator(); it.Next(); {
		k, v := it.Element()
		b.SetAttributeValue(k.AsString(), v)
	}
	return DecodeAndAssert(t, spec, string(hclwrite.Format(f.Bytes())), nil, asserts)
}

// DecodeAndAssert decodes a string representing a config, data, or content block
// into cty.Value according to given spec (i.e. respecting default values)
//
// Deprecated: use plugintest.NewTestDecoder
func DecodeAndAssert(t *testing.T, spec *dataspec.RootSpec, body string, dataCtx plugindata.Map, asserts diagtest.Asserts) (val *dataspec.Block) {
	t.Helper()
	var diags diagnostics.Diag
	var fm map[string]*hcl.File
	defer func() {
		t.Helper()
		asserts.AssertMatch(t, diags, fm)
	}()
	src := []byte("block {\n")
	src = append(src, body...)
	src = append(src, "\n}\n"...)
	filename := getFileName()
	f, diag := hclsyntax.ParseConfig(src, filename, hcl.InitialPos)
	if diags.Extend(diag) {
		return val
	}
	fm = map[string]*hcl.File{
		filename: f,
	}
	val, dgs := dataspec.DecodeAndEvalBlock(context.Background(), f.Body.(*hclsyntax.Body).Blocks[0], spec, dataCtx)
	if diags.Extend(dgs) {
		return val
	}
	return val
}

// Deprecated: use plugintest.NewTestDecoder
func Decode(t *testing.T, spec *dataspec.RootSpec, body string) (v *dataspec.Block, diags diagnostics.Diag) {
	t.Helper()
	src := []byte("block {\n")
	src = append(src, body...)
	src = append(src, "\n}\n"...)
	f, stdDiag := hclsyntax.ParseConfig(src, getFileName(), hcl.InitialPos)
	if diags.Extend(stdDiag) {
		return v, diags
	}

	v, diag := dataspec.DecodeAndEvalBlock(context.Background(), f.Body.(*hclsyntax.Body).Blocks[0], spec, nil)
	diags.Extend(diag)
	return v, diags
}
