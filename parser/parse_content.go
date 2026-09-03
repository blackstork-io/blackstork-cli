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

func parseContentBlock(
	ctx context.Context,
	blocksRegistry BlocksRegistry,
	execBlockDef *definitions.ExecBlockDef,
	refHist *utils.RefHistory,
) (res *definitions.ContentBlock, diags diagnostics.Diag) {
	execBlockDef = cloneExecBlockDef(execBlockDef)
	log := appctx.Log(ctx)
	log = log.With(
		"block_name", execBlockDef.Name(),
		"runner", execBlockDef.RunnerName(),
		"is_ref", execBlockDef.IsRef(),
	)
	log.DebugContext(ctx, "Parsing content block", "ref_hist", refHist.Size())

	blockName := execBlockDef.Name()
	if blockName == "" {
		blockName = makeNamePlaceholder()
	}

	res = &definitions.ContentBlock{
		Source:     execBlockDef,
		RunnerName: execBlockDef.RunnerName(),
		BlockName:  blockName,
	}

	// Parsing body
	body := execBlockDef.Block.Body

	validChildren := []string{
		definitions.BlockKindMeta,
		definitions.BlockKindConfig,
		definitions.BlockKindVars,
	}
	validChildrenSet := utils.SliceToSet(validChildren)

	var diag diagnostics.Diag

	var origVars *definitions.Vars

	var targetBlock *definitions.ContentBlock

	// Parsing the ref
	// If the block is a ref, inherit attributes from the base block.
	// The attributes are overridden if they are defined in this block.

	isRef := execBlockDef.IsRef()

	// Collect `base` value for ref and pop it to avoid validation errors
	// when checking runner-specific attrs later
	refBase, refBaseFound := utils.Pop(body.Attributes, definitions.AttrRefBase)

	switch {
	case isRef && !refBaseFound:
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Ref block is missing `base` argument",
			Subject:  body.MissingItemRange().Ptr(),
			Context:  &body.SrcRange,
		})
		return res, diags
	case !isRef && refBaseFound:
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Non-ref block contains `base` argument",
			Subject:  refBase.Range().Ptr(),
			Context:  &body.SrcRange,
		})
	case !isRef && !refBaseFound: // happy path, no ref
		// do nothing
	case isRef && refBaseFound: // happy path, a correct ref block
		baseEval, diag := parseRefBase(
			ctx,
			blocksRegistry,
			execBlockDef.Kind(),
			execBlockDef.Block.Body,
			refBase.Expr,
			refHist,
		)
		if diags.Extend(diag) {
			return res, diags
		}

		switch baseEval := baseEval.(type) {
		case *definitions.ContentBlock:

			targetBlock = baseEval

			// Replaces "ref" with proper runner name
			res.RunnerName = targetBlock.RunnerName

			// Replace the name with base block name if not set in the definition
			if execBlockDef.Name() == "" {
				res.BlockName = targetBlock.BlockName
			}

			// Update HCL body with missing attributes / blocks, no overrriding
			updateOrigBody(body, targetBlock.Source.Block.Body)

		default:
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid block reference - mismatched type of a reference block",
				Detail: fmt.Sprintf(
					"`%s ref` block references a block of a mismatched type `%s` in `base` argument",
					execBlockDef.Kind(),
					baseEval.GetSourceKind(),
				),
				Context: body.Range().Ptr(),
			})
			return res, diags
		}
	}

	// Collect `local_var` attr and pop it to avoid validation errors when
	// checking runner-specific attrs later
	localVar, _ := utils.Pop(body.Attributes, definitions.AttrLocalVar)

	// var origConfig evaluation.Configuration
	var configBlock *hclsyntax.Block

	newBlocks := []*hclsyntax.Block{}

	for _, block := range body.Blocks {

		if !utils.Contains(validChildrenSet, block.Type) {
			diags.Append(definitions.NewNestingDiag(
				execBlockDef.Block.Type,
				block,
				body,
				validChildren,
			))
			continue
		}

		switch block.Type {
		case definitions.BlockKindConfig:
			log.DebugContext(ctx, "Config sub-block found")
			if configBlock != nil {
				diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "More than one inline config block defined",
					Detail:   "Only one inline config block can be defined inside content block. Ignoring all config blocks after the first one.",
					Subject:  block.DefRange().Ptr(),
					Context:  execBlockDef.Block.Range().Ptr(),
				})
				continue
			}

			configBlock = block

		case definitions.BlockKindMeta:
			if res.Meta != nil {
				diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "More than one inline meta block defined",
					Detail:   "Only one inline meta block can be defined inside content block. Ignoring all meta blocks after the first one.",
					Subject:  block.DefRange().Ptr(),
					Context:  execBlockDef.Block.Range().Ptr(),
				})
				break
			}

			var meta definitions.MetaBlock
			if diags.Extend(gohcl.DecodeBody(block.Body, nil, &meta)) {
				break
			}
			res.Meta = &meta

		case definitions.BlockKindVars:
			if origVars != nil {
				diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "More than one inline vars block defined",
					Detail: fmt.Sprintf(
						"Only one inline `vars` block can be defined inside `%s` block. Ignoring all vars blocks after the first one",
						execBlockDef.Kind(),
						// origVars.DefRange().Filename,
						// origVars.DefRange().Start.Line,
					),
					Subject: block.DefRange().Ptr(),
					Context: execBlockDef.Block.Body.Range().Ptr(),
				})
				break
			}
			origVars, diag = ParseVars(ctx, block, localVar)
			if diags.Extend(diag) {
				origVars = nil
			}
		default:
			// only non-common blocks are kept, as this list is passed to the block's runner for initialization
			newBlocks = append(newBlocks, block)
		}
	}

	if diags.Extend(diag) {
		return res, diags
	}

	// assign only non-common blocks to the body's list
	body.Blocks = newBlocks

	if targetBlock != nil && targetBlock.Config != nil {
		res.Config = targetBlock.Config
	}
	if res.Meta == nil && targetBlock != nil {
		res.Meta = targetBlock.Meta
	}

	// Collect `config` attr and pop it to avoid validation errors when
	// checking runner-specific attrs later
	configAttr, _ := utils.Pop(body.Attributes, definitions.BlockKindConfig)
	if configAttr != nil {
		log.DebugContext(ctx, "Config attribute found")
	}

	if configBlock != nil || configAttr != nil {
		config, diag := parseExecBlockDefConfig(
			blocksRegistry,
			execBlockDef,
			configAttr,
			configBlock,
		)
		if diags.Extend(diag) {
			return res, diags
		}

		// Overwrite with orig config if set
		if config != nil {
			res.Config = config
		}
	}

	// Set default config if no config provided
	if res.Config == nil {
		if defaultCfg := blocksRegistry.GetDefaultBlockConfig(execBlockDef); defaultCfg != nil {
			// Apply default configs to non-refs only
			log.DebugContext(ctx, "Default config loaded for block")
			res.Config = defaultCfg
		} else {
			res.Config = nil
		}
	}

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

	if targetBlock != nil {
		res.Vars.Extend(targetBlock.Vars)
	}

	// Collect `is_included` and pop it to avoid validation errors when checking runner-specific attrs later
	if origIsIncluded, ok := utils.Pop(body.Attributes, definitions.AttrIsIncluded); ok {
		res.IsIncluded = origIsIncluded
	} else if targetBlock != nil && targetBlock.IsIncluded != nil {
		res.IsIncluded = targetBlock.IsIncluded
	}

	// Collect `required_vars` and pop it to avoid validation errors when checking runner-specific attrs later
	var requiredVarsCombined []*hclsyntax.Attribute
	if reqVarsAttr, found := utils.Pop(body.Attributes, definitions.AttrRequiredVars); found {
		requiredVarsCombined = append(requiredVarsCombined, reqVarsAttr)
	}

	// Combine required vars from origin and target
	if targetBlock != nil {
		requiredVarsCombined = append(requiredVarsCombined, targetBlock.RequiredVarsCombined...)
	}
	res.RequiredVarsCombined = requiredVarsCombined

	// Collect `depends_on` and pop it to avoid validation errors when checking runner-specific attrs later
	var dependsOnCombined []*hclsyntax.Attribute
	if depAttr, ok := utils.Pop(body.Attributes, definitions.AttrDependsOn); ok {
		dependsOnCombined = append(dependsOnCombined, depAttr)
	}
	if targetBlock != nil {
		dependsOnCombined = append(dependsOnCombined, targetBlock.DependsOnCombined...)
	}
	res.DependsOnCombined = dependsOnCombined

	return res, diags
}
