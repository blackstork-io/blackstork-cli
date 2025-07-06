package parser

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/parser/evaluation"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/utils"
)

func parseContentBlock(
	ctx context.Context,
	blocksRegistry BlocksRegistry,
	execBlockDef *definitions.ExecBlockDef,
	refHist *utils.RefHistory,
) (res *definitions.ContentBlock, diags diagnostics.Diag) {

	log := fabctx.GetLog(ctx)
	log.DebugContext(
		ctx, "Parsing a content block",
		"name", execBlockDef.Name(),
		"content_provider", execBlockDef.RunnerName(),
		"is_ref", execBlockDef.IsRef(),
		"ref_hist", refHist.Size(),
	)

	blockName := execBlockDef.Name()
	if blockName == "" {
		blockName = makeNamePlaceholder()
	}

	res = &definitions.ContentBlock{
		Source:          execBlockDef,
		BlockRunnerName: execBlockDef.RunnerName(),
		BlockName:       blockName,
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

	var origConfig evaluation.Configuration
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
		return
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
		baseEval, diag := parseRefBase(ctx, blocksRegistry, execBlockDef, refBase.Expr, refHist)
		if diags.Extend(diag) {
			return
		}

		switch baseEval := baseEval.(type) {
		case *definitions.ContentBlock:

			targetBlock = baseEval

			// Replaces "ref" with proper runner name
			res.BlockRunnerName = targetBlock.BlockRunnerName

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
			return
		}
	}

	// Collect `local_var` attr and pop it to avoid validation errors when
	// checking runner-specific attrs later
	localVar, _ := utils.Pop(body.Attributes, definitions.AttrLocalVar)

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
			if origConfig != nil {
				diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "More than one inline config block defined",
					Detail:   "Only one inline config block can be defined inside content block. Ignoring all config blocks after the first one.",
					Subject:  block.DefRange().Ptr(),
					Context:  execBlockDef.Block.Range().Ptr(),
				})
				break
			}
			// Collect `config` attr and pop it to avoid validation errors when
			// checking runner-specific attrs later
			configAttr, _ := utils.Pop(body.Attributes, definitions.BlockKindConfig)
			configBlock, diag := parseExecBlockDefConfig(blocksRegistry, execBlockDef, configAttr, block)
			if diags.Extend(diag) {
				break
			}
			origConfig = configBlock

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
						//origVars.DefRange().Filename,
						//origVars.DefRange().Start.Line,
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
		}
	}

	if diags.Extend(diag) {
		return
	}

	if targetBlock != nil && targetBlock.Config != nil {
		res.Config = targetBlock.Config
	}
	// Overwrite target config with orig config if set
	if origConfig != nil {
		res.Config = origConfig
	}
	// Set default config if no config provided
	if res.Config == nil {
		if defaultCfg := blocksRegistry.GetDefaultRunnerConfigForBlock(execBlockDef); defaultCfg != nil {
			// Apply default configs to non-refs only
			res.Config = defaultCfg
		} else {
			res.Config = &definitions.ConfigEmpty{
				ExecBlockDef: execBlockDef,
			}
		}
	}

	res.Vars = &definitions.Vars{}
	if targetBlock != nil {
		res.Vars.Extend(targetBlock.Vars)
	}
	if origVars == nil {
		origVars, diag = ParseVars(ctx, nil, localVar)
		if diags.Extend(diag) {
			origVars = nil
		}
	}
	res.Vars.Extend(origVars)

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
