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
	sourceBlockDefKind string,
	sourceBlockDefBody *hclsyntax.Body,
	base hcl.Expression,
	refHist *utils.RefHistory,
) (targetBlock definitions.RefTargetBlock, diags diagnostics.Diag) {
	log := appctx.Log(ctx)

	baseAttrVal := base.Range().Ptr()

	log.DebugContext(
		ctx, "Parsing ref block",
		"source_kind", sourceBlockDefKind,
		"refs_stack_size", refHist.Size(),
	)

	var baseBlockDef definitions.BlockDef

	switch sourceBlockDefKind {
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
			Summary:  "Referred block of invalid type",
			Detail:   fmt.Sprintf("Base block has invalid type `%s`", sourceBlockDefKind),
			Subject:  baseAttrVal,
			Context:  sourceBlockDefBody.Range().Ptr(),
		})
	}

	// becaise `ResolveRefBase` can return a typed pointer
	if baseBlockDef == nil || reflect.ValueOf(baseBlockDef).IsNil() {
		// Override all previous diags as irrelevant
		diags = diagnostics.Diag{
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Referred block not found",
				Detail: fmt.Sprintf(
					"Base block referred in %s is not found",
					baseAttrVal,
				),
				Subject: baseAttrVal,
				Context: sourceBlockDefBody.Range().Ptr(),
			},
		}
		return targetBlock, diags
	}

	if diags.HasErrors() {
		return targetBlock, diags
	}

	if sourceBlockDefKind != baseBlockDef.Kind() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid block reference: mismatched type of a source and a target block",
			Detail: fmt.Sprintf(
				"'%s ref' block references a block of a mismatched type `%s` in `base` argument",
				sourceBlockDefKind,
				baseBlockDef.Kind(),
			),
			Subject: baseAttrVal,
			Context: sourceBlockDefBody.Range().Ptr(),
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
			Detail:   "Block reference chains reached allowed max depth",
			Subject:  sourceBlockDefBody.Range().Ptr(),
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
		// if called here, there are no doc format blocks
		targetBlock, diag = parsePublishBlock(ctx, blocksRegistry, nil, publishDef, refHist)
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
