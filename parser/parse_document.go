package parser

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
)

func ParseDocument(
	ctx context.Context,
	blocksRegistry BlocksRegistry,
	docDef *definitions.DocumentDef,
) (doc *definitions.Document, diags diagnostics.Diag) {

	log := fabctx.GetLog(ctx)
	log = log.With("document", docDef.Name)

	body := docDef.Block.Body

	log.DebugContext(
		ctx, "Parsing a document template",
		"attributes_count", len(body.Attributes),
		"blocks_count", len(body.Blocks),
	)

	doc = &definitions.Document{}
	doc.Source = docDef

	if title := body.Attributes[definitions.AttrTitle]; title != nil {
		titleContent, diag := parseTitle(ctx, blocksRegistry, title)
		if !diag.Extend(diags) {
			doc.ContentTreeBlocks = append(doc.ContentTreeBlocks, titleContent)
		}
	}

	var origMeta *hcl.Range
	var varsBlock *hclsyntax.Block

	for _, block := range body.Blocks {
		log.DebugContext(ctx, "Parsing a block", "type", block.Type, "labels", block.Labels)

		switch block.Type {
		// Document-level blocks
		case definitions.BlockKindContent:
			blockDef, diag := definitions.DefineExecBlockDef(block, false)
			if diags.Extend(diag) {
				continue
			}
			var content definitions.ContentTreeBlock
			content, diags = parseContentBlock(ctx, blocksRegistry, blockDef, nil)
			doc.ContentTreeBlocks = append(doc.ContentTreeBlocks, content)

		case definitions.BlockKindData:
			blockDef, diag := definitions.DefineExecBlockDef(block, false)
			if diags.Extend(diag) {
				continue
			}
			var data *definitions.DataBlock
			data, diags = parseDataBlock(ctx, blocksRegistry, blockDef, nil)
			doc.DataBlocks = append(doc.DataBlocks, data)

		case definitions.BlockKindPublish:
			blockDef, diag := definitions.DefineExecBlockDef(block, false)
			if diags.Extend(diag) {
				continue
			}
			var publish *definitions.PublishBlock
			publish, diags = parsePublishBlock(ctx, blocksRegistry, blockDef, nil)
			doc.PublishBlocks = append(doc.PublishBlocks, publish)

		case definitions.BlockKindFormat:
			blockDef, diag := definitions.DefineExecBlockDef(block, false)
			if diags.Extend(diag) {
				continue
			}
			var format *definitions.FormatBlock
			format, diags = parseFormatBlock(ctx, blocksRegistry, blockDef, nil)
			doc.FormatBlocks = append(doc.FormatBlocks, format)

		case definitions.BlockKindVars:
			if varsBlock != nil {
				diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Vars block redefinition",
					Detail: fmt.Sprintf(
						"Only one `vars` block allowed in `%s` and one is already defined at %s:%d",
						docDef.Block.Type, varsBlock.DefRange().Filename, varsBlock.DefRange().Start.Line,
					),
					Subject: block.DefRange().Ptr(),
					Context: body.Range().Ptr(),
				})
				continue
			}
			varsBlock = block
		case definitions.BlockKindMeta:
			if origMeta != nil {
				diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Meta block redefinition",
					Detail: fmt.Sprintf(
						"Only one `meta` block allowed in `%s` and one is already defined at %s:%d",
						docDef.Block.Type, origMeta.Filename, origMeta.Start.Line,
					),
					Subject: block.DefRange().Ptr(),
					Context: body.Range().Ptr(),
				})
				continue
			}
			var meta definitions.MetaBlock
			if diags.Extend(gohcl.DecodeBody(block.Body, nil, &meta)) {
				continue
			}
			doc.Meta = &meta
			origMeta = block.DefRange().Ptr()
		case definitions.BlockKindSection:
			section, diag := definitions.DefineSectionDef(block, false)
			if diags.Extend(diag) {
				continue
			}
			parsedSection, diag := parseSection(ctx, blocksRegistry, section, nil)
			if diags.Extend(diag) {
				continue
			}
			doc.ContentTreeBlocks = append(doc.ContentTreeBlocks, parsedSection)

		case definitions.BlockKindDynamic:
			dynamic, diag := parseDynamic(ctx, blocksRegistry, block, nil)
			if diags.Extend(diag) {
				continue
			}
			doc.ContentTreeBlocks = append(doc.ContentTreeBlocks, dynamic)
		default:
			diags.Append(definitions.NewNestingDiag(
				docDef.Block.Type,
				block,
				body,
				[]string{
					definitions.BlockKindContent,
					definitions.BlockKindData,
					definitions.BlockKindMeta,
					definitions.BlockKindVars,
					definitions.BlockKindSection,
					definitions.BlockKindFormat,
					definitions.BlockKindPublish,
					definitions.BlockKindDynamic,
				},
			))
			continue
		}
	}

	// Extract `vars` block
	var diag diagnostics.Diag
	doc.Vars, diag = ParseVars(ctx, varsBlock, body.Attributes[definitions.AttrLocalVar])
	diags.Extend(diag)

	// Extract `required_vars` block
	if requiredVarsAttr := body.Attributes[definitions.AttrRequiredVars]; requiredVarsAttr != nil {
		diag := gohcl.DecodeExpression(requiredVarsAttr.Expr, nil, &doc.RequiredVars)
		diags.Extend(diag)
	}
	return
}
