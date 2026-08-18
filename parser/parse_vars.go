// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package parser

import (
	"context"
	"slices"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/deferred"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

var varsSpec = &dataspec.AttrSpec{
	Name: definitions.BlockKindVars,
	Type: plugindata.Encapsulated.CtyType(),
}

func ParseVars(ctx context.Context, block *hclsyntax.Block, localVar *hclsyntax.Attribute) (parsed *definitions.Vars, diags diagnostics.Diag) {
	if block == nil && localVar == nil {
		parsed = &definitions.Vars{}
		return parsed, diags
	}
	if block != nil && localVar != nil {
		localVarInVars := block.Body.Attributes[definitions.LocalVarName]
		if localVarInVars != nil {
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Local var redefinition",
				Detail:   "Local var is defined both in vars block and as a separate argument",
				Subject:  localVar.Range().Ptr(),
				Context:  block.Body.Range().Ptr(),
			})
		} else {
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Local var specified together with vars block",
				Detail: "It's recommended to use either `vars` block or `local_var`, not both at the same time. " +
					"You can define a variable named `local` in an existing `vars` block if needed.",
				Subject: localVar.Range().Ptr(),
			})
		}
	}

	var varCount int
	if block != nil {
		for _, subBlock := range block.Body.Blocks {
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Invalid nested block",
				Detail:   "`vars` block doesn't support nested blocks.",
				Subject:  subBlock.Range().Ptr(),
			})
		}
		varCount = len(block.Body.Attributes)
	}

	if localVar != nil {
		localVar.Name = "local"
		varCount++
	}
	evalCtx := deferred.WithQueryFuncs(appctx.GetEvalContext(ctx))
	vars := make([]*dataspec.Attr, 0, varCount)

	if block != nil {
		for _, attr := range block.Body.Attributes {
			val, diag := dataspec.DecodeAttr(evalCtx, attr, varsSpec)
			if !diags.Extend(diag) {
				vars = append(vars, val)
			}
		}
	}
	// ordered by definition
	slices.SortFunc(vars, func(a, b *dataspec.Attr) int {
		return a.NameRange.Start.Byte - b.NameRange.Start.Byte
	})
	if localVar != nil {
		// ordered last
		val, diag := dataspec.DecodeAttr(evalCtx, localVar, varsSpec)
		if !diags.Extend(diag) {
			vars = append(vars, val)
		}
	}
	parsed = &definitions.Vars{}
	parsed.Append(vars...)
	return parsed, diags
}
