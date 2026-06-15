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
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

func parseRefBase(
	ctx context.Context,
	blocksRegistry BlocksRegistry,
	sourceBlockDef definitions.RefSourceDef,
	base hcl.Expression,
	refHist *utils.RefHistory,
) (targetBlock definitions.RefTargetBlock, diags diagnostics.Diag) {
	log := appctx.Log(ctx)

	baseAttrVal := base.Range().Ptr()
	// 	baseVal, d := base.Value(&hcl.EvalContext{
	// 	})

	log.DebugContext(
		ctx, "Parsing ref block",
		"source_kind", sourceBlockDef.Kind(),
		//"base_ref", baseVal,
		"refs_stack_size", refHist.Size(),
	)

	var baseBlockDef definitions.BlockDef

	switch sourceBlockDef.Kind() {
	case definitions.BlockKindData,
		definitions.BlockKindContent,
		definitions.BlockKindFormat,
		definitions.BlockKindPublish:

		baseBlockDef, diags = blocksRegistry.ResolveRefBase(base, new(definitions.ExecBlockDef))
	case definitions.BlockKindSection:
		// `section` blocks
		baseBlockDef, diags = blocksRegistry.ResolveRefBase(base, new(definitions.SectionDef))
	default:
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid ref base block type",
			Detail:   fmt.Sprintf("Invalid ref base type `%s`", sourceBlockDef.Kind()),
			Subject:  baseAttrVal,
			Context:  sourceBlockDef.GetHCLBlock().Body.Range().Ptr(),
		})
	}

	// becaise `ResolveRefBase` can return a typed pointer
	if baseBlockDef == nil || reflect.ValueOf(baseBlockDef).IsNil() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Referred base block is not found",
			Detail:   fmt.Sprintf("Referred block of type `%s` is not found", sourceBlockDef.Kind()),
			Subject:  baseAttrVal,
			Context:  sourceBlockDef.GetHCLBlock().Body.Range().Ptr(),
		})
		return targetBlock, diags
	}

	if diags.HasErrors() {
		return targetBlock, diags
	}

	if sourceBlockDef.Kind() != baseBlockDef.Kind() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid block reference: mismatched type of a source and a target block",
			Detail: fmt.Sprintf(
				"'%s ref' block references a block of a mismatched type `%s` in `base` argument",
				sourceBlockDef.Kind(),
				baseBlockDef.Kind(),
			),
			Subject: baseAttrVal,
			Context: sourceBlockDef.GetHCLBlock().Body.Range().Ptr(),
		})
		return targetBlock, diags
	}

	if refHist == nil {
		refHist = utils.NewRefHistory()
	}
	if !refHist.IsRefAllowed() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reached the maximum reference depth",
			Detail:   "Content reference chains are limited to 10 references",
			Subject:  sourceBlockDef.DefRange().Ptr(),
			Extra:    diagnostics.NewTracebackExtra(),
		})
		return targetBlock, diags
	}
	refHist.Add(baseAttrVal)
	defer refHist.Pop()

	var diag diagnostics.Diag

	switch baseBlockDef.Kind() {
	case definitions.BlockKindContent:
		contentBlockDef := baseBlockDef.(*definitions.ExecBlockDef)
		targetBlock, diag = parseContentBlock(ctx, blocksRegistry, contentBlockDef, refHist)
	case definitions.BlockKindData:
		dataBlockDef := baseBlockDef.(*definitions.ExecBlockDef)
		targetBlock, diag = ParseDataBlock(ctx, blocksRegistry, dataBlockDef, refHist)
	case definitions.BlockKindSection:
		sectionDef := baseBlockDef.(*definitions.SectionDef)
		targetBlock, diag = ParseSection(ctx, blocksRegistry, sectionDef, refHist)
	case definitions.BlockKindFormat:
		formatDef := baseBlockDef.(*definitions.ExecBlockDef)
		targetBlock, diag = ParseFormatBlock(ctx, blocksRegistry, formatDef, refHist)
	case definitions.BlockKindPublish:
		publishDef := baseBlockDef.(*definitions.ExecBlockDef)
		targetBlock, diag = parsePublishBlock(ctx, blocksRegistry, publishDef, refHist)
	}
	diags.Extend(diag)
	return targetBlock, diags
}

func updateOrigBody(orig, base *hclsyntax.Body) {
	// remove `base` ref attribute
	utils.Pop(orig.Attributes, definitions.AttrRefBase)

	// Adopt only attributes missing in the orig block
	for k, v := range base.Attributes {
		if _, found := orig.Attributes[k]; found {
			continue
		}
		orig.Attributes[k] = v
	}
	// Adopt only blocks missing in the orig blocks
	origBlocks := make(map[string]struct{}, len(orig.Blocks))
	for _, b := range orig.Blocks {
		origBlocks[hclBlockKey(b)] = struct{}{}
	}
	for _, b := range base.Blocks {
		if _, found := origBlocks[hclBlockKey(b)]; found {
			continue
		}
		orig.Blocks = append(orig.Blocks, b)
	}
}
