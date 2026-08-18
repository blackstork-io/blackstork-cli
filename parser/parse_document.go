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
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

func ParseDocument(
	ctx context.Context,
	blocksRegistry BlocksRegistry,
	docDef *definitions.DocumentDef,
) (doc *definitions.Document, diags diagnostics.Diag) {
	log := appctx.Log(ctx)
	log = log.With("document", docDef.Name)

	hclBlock := docDef.GetHCLBlock()
	body := hclBlock.Body

	log.DebugContext(
		ctx, "Parsing doc definition",
		"attributes_count", len(body.Attributes),
		"blocks_count", len(body.Blocks),
	)

	doc = &definitions.Document{}
	doc.Source = docDef

	if title := body.Attributes[definitions.AttrTitle]; title != nil {
		titleContent, diag := parseTitle(ctx, blocksRegistry, title)
		if !diags.Extend(diag) {
			doc.Title = titleContent
		}
	}

	var origMeta *hcl.Range
	var varsBlock *hclsyntax.Block

	for _, block := range body.Blocks {
		switch block.Type {
		// Document-level blocks
		case definitions.BlockKindContent:
			blockDef, diag := definitions.DefineExecBlockDef(block, false)
			if diags.Extend(diag) {
				continue
			}
			var content definitions.ContentTreeBlock
			content, diag = parseContentBlock(ctx, blocksRegistry, blockDef, nil)
			if diags.Extend(diag) {
				continue
			}
			doc.ContentTreeBlocks = append(doc.ContentTreeBlocks, content)

		case definitions.BlockKindData:
			blockDef, diag := definitions.DefineExecBlockDef(block, false)
			if diags.Extend(diag) {
				continue
			}
			var data *definitions.DataBlock
			data, diag = ParseDataBlock(ctx, blocksRegistry, blockDef, nil)
			if diags.Extend(diag) {
				continue
			}
			doc.DataBlocks = append(doc.DataBlocks, data)

		case definitions.BlockKindFormat:
			blockDef, diag := definitions.DefineExecBlockDef(block, false)
			if diags.Extend(diag) {
				continue
			}
			var format *definitions.FormatBlock
			format, diag = ParseFormatBlock(ctx, blocksRegistry, blockDef, nil)
			if diags.Extend(diag) {
				continue
			}
			doc.FormatBlocks = append(doc.FormatBlocks, format)

		case definitions.BlockKindVars:
			if varsBlock != nil {
				diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Vars block redefinition",
					Detail: fmt.Sprintf(
						"Only one `vars` block allowed in `%s` and one is already defined at %s:%d",
						hclBlock.Type,
						varsBlock.DefRange().Filename,
						varsBlock.DefRange().Start.Line,
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
						hclBlock.Type, origMeta.Filename, origMeta.Start.Line,
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

		case definitions.BlockKindInput:

			if len(block.Labels) != 1 {
				diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Input blocks must be named",
					Detail:   `Anonymous input blocks are not supported. Each input block must have a name to be used as a reference to its value.`,
					Subject:  block.DefRange().Ptr(),
					Context:  body.Range().Ptr(),
				})
				continue
			}

			name := block.Labels[0]

			var input definitions.InputBlock
			if diags.Extend(gohcl.DecodeBody(block.Body, nil, &input)) {
				log.DebugContext(ctx, "Error while unpacking input block", "name", name)
				continue
			}
			input.Name = name
			input.Block = block
			doc.Inputs = append(doc.Inputs, &input)

		case definitions.BlockKindSection:
			section, diag := definitions.DefineSectionDef(block, false)
			if diags.Extend(diag) {
				continue
			}
			parsedSection, diag := ParseSection(ctx, blocksRegistry, section, nil)
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
		case definitions.BlockKindPublish:
			// parse `publish` blocks after all other were parsed,
			// as `format_ref` attr might reference in-document `format` blocks
		default:
			diags.Append(definitions.NewNestingDiag(
				hclBlock.Type,
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

	for _, block := range body.Blocks {
		switch block.Type {
		case definitions.BlockKindPublish:
			blockDef, diag := definitions.DefineExecBlockDef(block, false)
			if diags.Extend(diag) {
				continue
			}
			var publish *definitions.PublishBlock
			publish, diag = parsePublishBlock(ctx, blocksRegistry, doc, blockDef, nil)
			if diags.Extend(diag) {
				continue
			}
			doc.PublishBlocks = append(doc.PublishBlocks, publish)
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
	return doc, diags
}
