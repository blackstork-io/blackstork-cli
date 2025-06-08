package parser

import (
	"context"
	"fmt"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/utils"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func (db *DefinedBlocks) parseRefBase(
	ctx context.Context,
	sourceBlockDef definitions.RefSourceDef,
	base hcl.Expression,
	refHist *utils.RefHistory,
) (targetBlock definitions.RefTargetBlock, diags diagnostics.Diag) {

	log := fabctx.GetLog(ctx)

	baseAttrVal := base.Range().Ptr()

	log.DebugContext(
		ctx, "Parsing a ref block",
		"source_kind", sourceBlockDef.Kind(),
		"base_ref", baseAttrVal,
		"ref_hist", refHist.Size(),
	)

	var baseBlockDef definitions.BlockDef

	switch sourceBlockDef.Kind() {
	case definitions.BlockKindData,
		definitions.BlockKindContent,
		definitions.BlockKindFormat,
		definitions.BlockKindPublish:
		baseBlockDef, diags = ResolveWithDefined[*definitions.ExecBlockDef](db, base)
	case definitions.BlockKindSection:
		// `section` blocks
		baseBlockDef, diags = ResolveWithDefined[*definitions.SectionDef](db, base)
	default:
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid ref source block type",
			Detail:   fmt.Sprintf("Invalid ref source type `%s`", sourceBlockDef.Kind()),
			Subject:  baseAttrVal,
			Context:  sourceBlockDef.GetHCLBlock().Body.Range().Ptr(),
		})
	}

	if diags.HasErrors() {
		return
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
		return
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
		return
	}
	refHist.Add(baseAttrVal)
	defer refHist.Pop()

	var diag diagnostics.Diag

	switch baseBlockDef.Kind() {
	case definitions.BlockKindContent:
		contentBlockDef := baseBlockDef.(*definitions.ExecBlockDef)
		targetBlock, diag = db.ParseContentBlock(ctx, contentBlockDef, refHist)
	case definitions.BlockKindData:
		dataBlockDef := baseBlockDef.(*definitions.ExecBlockDef)
		targetBlock, diag = db.ParseDataBlock(ctx, dataBlockDef, refHist)
	case definitions.BlockKindSection:
		sectionDef := baseBlockDef.(*definitions.SectionDef)
		targetBlock, diag = db.ParseSection(ctx, sectionDef, refHist)
	}
	diags.Extend(diag)
	return
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
