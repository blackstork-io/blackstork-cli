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
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

func ParseSection(
	ctx context.Context,
	blocksRegistry BlocksRegistry,
	sectionDef *definitions.SectionDef,
	refHist *utils.RefHistory,
) (res *definitions.Section, diags diagnostics.Diag) {
	log := appctx.Log(ctx)
	log = log.With("section", sectionDef.Name())
	log.DebugContext(
		ctx, "Parsing section block",
		"is_ref", sectionDef.IsRef(),
		"ref_hist", refHist.Size(),
	)

	// Use a placeholder name if needed
	blockName := sectionDef.Name()
	if blockName == "" {
		blockName = makeNamePlaceholder()
	}

	res = &definitions.Section{
		Source:    sectionDef,
		BlockName: blockName,
	}

	// Parsing body
	body := sectionDef.Block.Body

	var diag diagnostics.Diag
	var validChildren []string

	isRef := sectionDef.IsRef()
	refBase, refBaseFound := body.Attributes[definitions.AttrRefBase]

	var targetSection *definitions.Section

	switch {
	case isRef && !refBaseFound:
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Ref block is missing `base` argument",
			Subject:  body.MissingItemRange().Ptr(),
			Context:  &body.SrcRange,
		})
		return nil, diags
	case !isRef && refBaseFound:
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Non-ref block contains `base` argument",
			Subject:  refBase.Range().Ptr(),
			Context:  &body.SrcRange,
		})
	case !isRef && !refBaseFound: // happy path, no ref
		validChildren = []string{
			definitions.BlockKindContent,
			definitions.BlockKindMeta,
			definitions.BlockKindSection,
			definitions.BlockKindVars,
			definitions.BlockKindDynamic,
		}
	case isRef && refBaseFound: // happy path, a correct ref block
		validChildren = []string{
			definitions.BlockKindContent,
			definitions.BlockKindMeta,
			definitions.BlockKindSection,
			definitions.BlockKindVars,
			definitions.BlockKindDynamic,
		}
		targetBlock, diag := parseRefBase(
			ctx,
			blocksRegistry,
			sectionDef.Kind(),
			sectionDef.Block.Body,
			refBase.Expr,
			refHist,
		)
		if diags.Extend(diag) {
			break
		}
		targetSection = targetBlock.(*definitions.Section)

		// Replace the name with base block name if not set in the definition
		if res.Source.Name() == "" {
			res.BlockName = targetSection.BlockName
		}
	}

	if diags.HasErrors() {
		return nil, diags
	}

	if title := body.Attributes["title"]; title != nil {
		titleContent, diag := parseTitle(ctx, blocksRegistry, title)
		if !diags.Extend(diag) {
			res.Title = titleContent
		}
	} else if targetSection != nil && targetSection.Title != nil {
		res.Title = targetSection.Title
	}

	// Prepend the ref base content blocks
	if targetSection != nil {
		res.Content = targetSection.Content
	}

	var origVars *definitions.Vars
	localVar := body.Attributes[definitions.AttrLocalVar]

	validChildrenSet := utils.SliceToSet(validChildren)

	newBlocks := []*hclsyntax.Block{}

	for _, block := range body.Blocks {
		if !utils.Contains(validChildrenSet, block.Type) {
			log.WarnContext(
				ctx, "Section contains an invalid block",
				"block_type", block.Type,
				"block_labels", block.Labels,
			)
			diags.Append(definitions.NewNestingDiag(
				sectionDef.Block.Type,
				block,
				body,
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
			content, diag := parseContentBlock(ctx, blocksRegistry, contentDef, refHist)
			if diags.Extend(diag) {
				continue
			}
			res.Content = append(res.Content, content)

		case definitions.BlockKindMeta:
			if res.Meta != nil {
				diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "More than one inline meta block defined",
					Detail:   "Only one inline meta block can be defined inside content block. Ignoring all meta blocks after the first one.",
					// Detail: fmt.Sprintf(
					// 	"Only one `meta` block allowed in `%s` and one is already defined at %s:%d",
					// 	sectionDef.Block.Type,
					// 	// origMeta.Filename,
					// 	// origMeta.Start.Line,
					// ),
					Subject: block.DefRange().Ptr(),
					Context: body.Range().Ptr(),
				})
				continue
			}
			var meta definitions.MetaBlock
			if diags.Extend(gohcl.DecodeBody(block.Body, nil, &meta)) {
				break
			}
			res.Meta = &meta

		case definitions.BlockKindVars:
			if origVars != nil {
				diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Vars block redefinition",
					Detail: fmt.Sprintf(
						"Only one inline `vars` block can be defined inside `%s` block. Ignoring all vars blocks after the first one",
						sectionDef.Block.Type,
						// block.DefRange().Filename,
						// block.DefRange().Start.Line,
					),
					Subject: block.DefRange().Ptr(),
					Context: body.Range().Ptr(),
				})
				break
			}

			origVars, diag = ParseVars(ctx, block, localVar)
			diags.Extend(diag)

		case definitions.BlockKindSection:
			subSectionDef, diag := definitions.DefineSectionDef(block, false)
			if diags.Extend(diag) {
				continue
			}
			subSection, diag := ParseSection(ctx, blocksRegistry, subSectionDef, refHist)
			if diags.Extend(diag) {
				continue
			}
			res.Content = append(res.Content, subSection)
		case definitions.BlockKindDynamic:
			dynamic, diag := parseDynamic(ctx, blocksRegistry, block, refHist)
			if diags.Extend(diag) {
				continue
			}
			res.Content = append(res.Content, dynamic)
		default:
			// only non-common blocks are kept
			newBlocks = append(newBlocks, block)
		}
	}

	if diags.Extend(diag) {
		return res, diags
	}

	// assign only non-common blocks to the body's list
	body.Blocks = newBlocks

	res.Vars = &definitions.Vars{}

	// Process original vars first

	if origVars == nil {
		origVars, diag = ParseVars(ctx, nil, localVar)
		if diags.Extend(diag) {
			origVars = nil
		}
	}
	res.Vars.Extend(origVars)

	// Apply target block vars after

	if targetSection != nil {
		res.Vars.Extend(targetSection.Vars)
	}

	// Collect `required_vars` and pop it to avoid validation errors when checking runner-specific attrs later
	var requiredVarsCombined []*hclsyntax.Attribute
	if reqVarsAttr, found := utils.Pop(body.Attributes, definitions.AttrRequiredVars); found {
		requiredVarsCombined = append(requiredVarsCombined, reqVarsAttr)
	}

	// Combine required vars from origin and target
	if targetSection != nil {
		requiredVarsCombined = append(requiredVarsCombined, targetSection.RequiredVarsCombined...)
	}
	res.RequiredVarsCombined = requiredVarsCombined

	// Collect `depends_on` and pop it to avoid validation errors when checking runner-specific attrs later
	var dependsOnCombined []*hclsyntax.Attribute
	if depAttr, ok := utils.Pop(body.Attributes, definitions.AttrDependsOn); ok {
		dependsOnCombined = append(dependsOnCombined, depAttr)
	}
	if targetSection != nil {
		dependsOnCombined = append(dependsOnCombined, targetSection.DependsOnCombined...)
	}
	res.DependsOnCombined = dependsOnCombined

	if origIsIncluded, found := body.Attributes[definitions.AttrIsIncluded]; found {
		res.IsIncluded = origIsIncluded
	} else if targetSection != nil && targetSection.IsIncluded != nil {
		res.IsIncluded = targetSection.IsIncluded
	}

	if res.Title == nil && targetSection != nil {
		res.Title = targetSection.Title
	}

	if res.Meta == nil && targetSection != nil {
		res.Meta = targetSection.Meta
	}

	return res, diags
}
