package parser

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/utils"
)

func parseDynamic(
	ctx context.Context,
	blocksRegistry BlocksRegistry,
	block *hclsyntax.Block,
	refHist *utils.RefHistory,
) (parsed *definitions.Dynamic, diags diagnostics.Diag) {

	log := fabctx.GetLog(ctx)
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
			parsedSubSection, diag := parseSection(ctx, blocksRegistry, subSection, refHist)
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
		}
	}
	if len(res.Children) == 0 {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "No content found in the dynamic block",
			Detail:   "Dynamic block without any content can be removed",
			Subject:  block.DefRange().Ptr(),
		})
	}
	parsed = &res
	return
}
