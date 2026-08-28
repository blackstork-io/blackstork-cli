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

func ParseDataBlock(
	ctx context.Context,
	blocksRegistry BlocksRegistry,
	execBlockDef *definitions.ExecBlockDef,
	refHist *utils.RefHistory,
) (parsed *definitions.DataBlock, diags diagnostics.Diag) {
	log := appctx.Log(ctx)
	log = log.With(
		"block_name", execBlockDef.Name(),
		"runner", execBlockDef.RunnerName(),
		"is_ref", execBlockDef.IsRef(),
	)
	log.DebugContext(ctx, "Parsing data block", "ref_hist", refHist.Size())

	res := definitions.DataBlock{
		Source:     execBlockDef,
		RunnerName: execBlockDef.RunnerName(),
		BlockName:  execBlockDef.Name(),
	}

	validChildren := []string{
		definitions.BlockKindMeta,
		definitions.BlockKindConfig,
	}
	validChildrenSet := utils.SliceToSet(validChildren)

	// Parsing body
	body := execBlockDef.Block.Body

	var targetBlock *definitions.DataBlock

	// Parsing the ref
	// If the block is a ref, inherit attributes from the base block.
	// The attributes are overridden if they are defined in this block.

	isRef := execBlockDef.IsRef()

	// Collect `base` value for ref and pop it to avoid validation errors
	// when checking runner-specific attrs later
	refBase, refBaseFound := utils.Pop(body.Attributes, definitions.AttrRefBase)

	switch {
	case isRef && !refBaseFound:
		log.ErrorContext(ctx, "No `base` attribute found in ref data block")
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Ref block is missing `base` attribute",
			Subject:  body.MissingItemRange().Ptr(),
			Context:  &body.SrcRange,
		})
		return parsed, diags
	case !isRef && refBaseFound:
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Normal non-ref block contains `base` attribute",
			Subject:  refBase.Range().Ptr(),
			Context:  &body.SrcRange,
		})
	case !isRef && !refBaseFound: // happy path, no ref
	case isRef && refBaseFound: // happy path, ref present
		baseEval, diag := parseRefBase(
			ctx,
			blocksRegistry,
			execBlockDef.Kind(),
			execBlockDef.Block.Body,
			refBase.Expr,
			refHist,
		)
		if diags.Extend(diag) {
			return parsed, diags
		}

		switch baseEval := baseEval.(type) {
		case *definitions.DataBlock:

			targetBlock = baseEval

			// replaces "ref" with actual name
			res.RunnerName = targetBlock.RunnerName

			if res.BlockName == "" {
				res.BlockName = targetBlock.BlockName
			}

			// updateOrigBody(invocation.Body, baseDataBlock.Invocation.Body)
			updateOrigBody(body, targetBlock.Source.Block.Body)
		default:
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid block reference - mismatched type of a reference block",
				Detail: fmt.Sprintf(
					"'%s ref' block references a block of a mismatched type `%s` in `base` attribute",
					execBlockDef.Kind(),
					baseEval.GetSourceKind(),
				),
				Context: body.Range().Ptr(),
			})
			return parsed, diags
		}
	}

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
					Detail:   "Only one inline config block allowed. Ignoring all config blocks after the first one.",
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
					Detail:   "Only one inline meta block can be defined. Ignoring all meta blocks after the first one.",
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
		default:
			// only non-common blocks are kept, as this list is passed to the block's runner for initialization
			newBlocks = append(newBlocks, block)
		}
	}

	if diags.HasErrors() {
		return parsed, diags
	}

	// assign only non-common blocks to the body's list
	body.Blocks = newBlocks

	if targetBlock != nil && targetBlock.Config != nil {
		res.Config = targetBlock.Config
	}

	// Collect `config` attr and pop it to avoid validation errors when
	// checking runner-specific attrs later
	configAttr, _ := utils.Pop(body.Attributes, definitions.BlockKindConfig)
	if configAttr != nil {
		log.DebugContext(ctx, "Config attribute found")
	}

	config, diag := parseExecBlockDefConfig(blocksRegistry, execBlockDef, configAttr, configBlock)
	if diags.Extend(diag) {
		log.ErrorContext(ctx, "Error while parsing config attr or inline config block", "err", diag)
		return parsed, diags
	}

	// Overwrite target config with orig config if set
	if config != nil {
		res.Config = config
	}

	if res.Config == nil {
		if defaultCfg := blocksRegistry.GetDefaultBlockConfig(execBlockDef); defaultCfg != nil {
			// Apply default configs to non-refs only
			res.Config = defaultCfg
			log.DebugContext(ctx, "Using default config for data block as no overrides provided")
		} else {
			res.Config = nil
		}
	}

	return &res, diags
}
