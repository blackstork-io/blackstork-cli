package parser

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/parser/evaluation"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/utils"
)

func parseDataBlock(
	ctx context.Context,
	blocksRegistry BlocksRegistry,
	execBlockDef *definitions.ExecBlockDef,
	refHist *utils.RefHistory,
) (parsed *definitions.DataBlock, diags diagnostics.Diag) {

	log := fabctx.GetLog(ctx)
	log.DebugContext(
		ctx, "Parsing a data block",
		"name", execBlockDef.Name(),
		"data_source", execBlockDef.RunnerName(),
		"is_ref", execBlockDef.IsRef(),
		"ref_hist", refHist.Size(),
	)

	res := definitions.DataBlock{
		Source:          execBlockDef,
		BlockRunnerName: execBlockDef.RunnerName(),
		BlockName:       execBlockDef.Name(),
	}

	validChildren := []string{
		definitions.BlockKindMeta,
		definitions.BlockKindConfig,
	}
	validChildrenSet := utils.SliceToSet(validChildren)

	// Parsing body
	body := execBlockDef.Block.Body

	var targetBlock *definitions.DataBlock

	refBase, refBaseFound := body.Attributes[definitions.AttrRefBase]
	isRef := execBlockDef.IsRef()

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
	case isRef && refBaseFound: // happy path, ref present
		baseEval, diag := parseRefBase(ctx, blocksRegistry, execBlockDef, refBase.Expr, refHist)
		if diags.Extend(diag) {
			return
		}

		switch baseEval := baseEval.(type) {
		case *definitions.DataBlock:

			targetBlock = baseEval

			// replaces "ref" with actual name
			res.BlockRunnerName = targetBlock.BlockRunnerName

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
					"'%s ref' block references a block of a mismatched type `%s` in `base` argument",
					execBlockDef.Kind(),
					baseEval.GetSourceKind(),
				),
				Context: body.Range().Ptr(),
			})
			return
		}
	}

	var origConfig evaluation.Configuration

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
					Detail:   "Only one inline config block can be defined. Ignoring all config blocks after the first one.",
					Subject:  block.DefRange().Ptr(),
					Context:  execBlockDef.Block.Range().Ptr(),
				})
				break
			}
			// collect the config if provided inside the block
			configAttr, _ := body.Attributes[definitions.BlockKindConfig]
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
		}
	}

	if diags.HasErrors() {
		return
	}

	// FIXME: why we reasign the body back? was `Body` value modified somehow?
	// execBlockDef.Block.Body = body
	// FIXME: WHY DO WE NEED THIS INVOCATION?
	// 	invocation := &evaluation.BlockInvocation{
	// 		Block: execBlockDef.Block,
	// 	}

	if targetBlock != nil && targetBlock.Config != nil {
		res.Config = targetBlock.Config
	}
	// Overwrite target config with orig config if set
	if origConfig != nil {
		res.Config = origConfig
	}

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

	// res.Invocation = invocation
	return &res, diags
}
