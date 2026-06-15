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

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

func parseDynamic(
	ctx context.Context,
	blocksRegistry BlocksRegistry,
	block *hclsyntax.Block,
	refHist *utils.RefHistory,
) (parsed *definitions.Dynamic, diags diagnostics.Diag) {
	log := appctx.Log(ctx)
	log.DebugContext(
		ctx, "Parsing a dynamic block",
		"labels", block.Labels,
		"ref_hist", refHist.Size(),
	)

	def := &definitions.DynamicDef{
		Block: block,
	}

	body := block.Body

	blockName := def.Name()
	if blockName == "" {
		blockName = makeNamePlaceholder()
	}

	res := definitions.Dynamic{
		Source:    def,
		BlockName: blockName,
	}
	res.Items, _ = utils.Pop(body.Attributes, definitions.AttrDynamicItems)

	if res.Items == nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Dynamic block without items",
			Detail:   fmt.Sprintf("Dynamic block must have an attribute %q", definitions.AttrDynamicItems),
			Subject:  block.DefRange().Ptr(),
		})
	}

	if depAttr, ok := utils.Pop(body.Attributes, definitions.AttrDependsOn); ok {
		res.DependsOn = depAttr
	}

	for k, v := range block.Body.Attributes {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Unsupported attribute",
			Detail:   fmt.Sprintf("Dynamic block does not support attribute %q, it will be ignored", k),
			Subject:  &v.NameRange,
		})
	}

	validChildren := []string{
		definitions.BlockKindContent,
		definitions.BlockKindSection,
		definitions.BlockKindDynamic,
	}
	validChildrenSet := utils.SliceToSet(validChildren)

	newBlocks := []*hclsyntax.Block{}

	for _, block := range block.Body.Blocks {
		if !utils.Contains(validChildrenSet, block.Type) {
			diags.Append(definitions.NewNestingDiag(
				block.Type,
				block,
				block.Body,
				validChildren,
			))
			continue
		}
		switch block.Type {
		case definitions.BlockKindContent:
			contentDef, diag := definitions.DefineExecBlockDef(block, false)
			if diags.Extend(diag) {
				continue
			}
			var content definitions.ContentTreeBlock
			content, diag = parseContentBlock(ctx, blocksRegistry, contentDef, refHist)
			if diags.Extend(diag) {
				continue
			}
			res.Children = append(res.Children, content)
		case definitions.BlockKindSection:
			subSection, diag := definitions.DefineSectionDef(block, false)
			if diags.Extend(diag) {
				continue
			}
			parsedSubSection, diag := ParseSection(ctx, blocksRegistry, subSection, refHist)
			if diags.Extend(diag) {
				continue
			}
			res.Children = append(res.Children, parsedSubSection)
		case definitions.BlockKindDynamic:
			subDynamic, diag := parseDynamic(ctx, blocksRegistry, block, refHist)
			if diags.Extend(diag) {
				continue
			}
			res.Children = append(res.Children, subDynamic)
		default:
			// only non-common blocks are kept
			newBlocks = append(newBlocks, block)
		}
	}

	// assign only non-common blocks to the body's list
	body.Blocks = newBlocks

	if len(res.Children) == 0 {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "No content found in the dynamic block",
			Detail:   "Dynamic block without any content can be removed",
			Subject:  block.DefRange().Ptr(),
		})
	}
	parsed = &res
	return parsed, diags
}
