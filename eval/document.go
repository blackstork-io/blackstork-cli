// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package eval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

type Document struct {
	Source *definitions.Document
	meta   *definitions.MetaBlock

	Inputs []*definitions.InputBlock

	Vars         *definitions.Vars
	RequiredVars []string
	DataBlocks   []*PluginDataAction

	Title             ContentTreeEvalBlock
	ContentTreeBlocks []ContentTreeEvalBlock

	PublishBlocks []*PluginPublishAction
	FormatBlocks  []*PluginFormatAction

	RequestedFormat *PluginFormatAction
}

func (doc *Document) FetchData(ctx context.Context) (plugindata.Map, diagnostics.Diag) {
	return executeDataBlocksAsync(ctx, doc, nil)
}

func (doc *Document) FetchDataWithPath(
	ctx context.Context,
	path []string,
) (plugindata.Data, diagnostics.Diag) {
	return executeDataBlocksAsync(ctx, doc, path)
}

func (doc *Document) GetTemplateName() string {
	return doc.Source.Source.Name
}

func (doc *Document) GetName() string {
	return fmt.Sprintf("%s.%s", definitions.BlockKindDocument, doc.GetTemplateName())
}

func (doc *Document) Meta() plugindata.Map {
	if doc.meta == nil {
		return plugindata.Map{}
	}
	return doc.meta.AsPluginData()
}

func GetFrame(block ContentTreeEvalBlock) *FrameNode {
	node := &FrameNode{
		ID:  block.ID(),
		Key: block.EvalKey(),
	}

	switch block := block.(type) {
	case *PluginContentAction:
	case *Section:
		if block.title != nil {
			node.Title = GetFrame(block.title)
		}
	case *Dynamic:
	default:
	}
	return node
}

func (doc *Document) GetFrame() *DocFrame {
	var title *FrameNode
	if doc.Title != nil {
		title = GetFrame(doc.Title)
	}

	children := []*FrameNode{}
	for _, block := range doc.ContentTreeBlocks {
		if utils.IsNil(block) {
			panic("GOT TYPED NIL :(")
		}
		children = append(children, GetFrame(block))
	}

	return &DocFrame{
		Title:    title,
		Children: children,
	}
}

func (doc *Document) RenderContent(
	ctx context.Context,
	dataCtx plugindata.Map,
	requiredTags []string,
) (*plugin.ContentSection, plugindata.Map, diagnostics.Diag) {
	log := appctx.Log(ctx)
	log.InfoContext(ctx, "Processing document template")

	// Eval and add `vars` and other keys to `dataCtx`
	diag := applyBlockDataToDataCtx(
		ctx,
		"document",
		doc.GetTemplateName(),
		definitions.BlockKindDocument,
		doc.Vars,
		doc.RequiredVars,
		doc.Meta(),
		"",
		0,
		dataCtx,
	)
	if diag.HasErrors() {
		log.ErrorContext(ctx, "Error while applying data to the context", "err", diag.Error())
		return nil, nil, diag
	}

	log.DebugContext(ctx, "Executing content blocks")

	result, diag := executeContentBlocksAsync(ctx, doc, requiredTags, dataCtx)
	if diag.HasErrors() {
		return nil, nil, diag
	}

	log.DebugContext(ctx, "Post-processing content blocks")

	// Post-process to fill in unique blocks, like TOC
	result, err := postExecProcessContentBlocks(ctx, result)
	if err != nil {
		diag.AppendErr(err, "Error while post-processing content blocks")
		return nil, nil, diag
	}

	return result, dataCtx, nil
}

func (doc *Document) FormatContent(
	ctx context.Context,
	content *plugin.ContentSection,
	data plugindata.Map,
	formatAction *PluginFormatAction,
) (_ *plugin.FormattedContent, diags diagnostics.Diag) {
	contentData := content.AsData()
	return formatAction.Execute(ctx, data, contentData)
}

func (doc *Document) Publish(
	ctx context.Context,
	content plugin.Content,
	formattedContents map[*PluginFormatAction]*plugin.FormattedContent,
	data plugindata.Data,
) (diags diagnostics.Diag) {
	log := appctx.Log(ctx)
	log = log.With("document", doc.GetTemplateName())

	docData := plugindata.Map{
		definitions.BlockKindContent: content.AsData(),
		definitions.BlockKindMeta:    doc.Meta(),
	}
	dataCtx := plugindata.Map{
		definitions.BlockKindData:     data,
		definitions.BlockKindDocument: docData,
	}

	if len(doc.PublishBlocks) == 0 {
		log.WarnContext(ctx, "No publish blocks found")
		return diags
	}

	for _, block := range doc.PublishBlocks {

		formattedContent, ok := formattedContents[block.Format]
		if !ok {
			log.ErrorContext(
				ctx, "Formatted content for publish block not found",
				"publish_block", block.FullBlockName(),
			)
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Formatted content not found for publish block",
				Detail: fmt.Sprintf(
					"Content from format `%s` not found for publish block `%s`",
					block.Format.FullBlockName(),
					block.FullBlockName(),
				),
			}}
		}

		log.DebugContext(
			ctx, "Executing publish block",
			"publish_block", block.FullBlockName(),
		)

		documentName := doc.GetTemplateName()
		diag := block.Publish(ctx, dataCtx, content, formattedContent, documentName)
		if diag != nil {
			diags.Extend(diag)
		}
	}
	return diags
}

func (doc *Document) PublishAction(
	ctx context.Context,
	content plugin.Content,
	formattedContents map[*PluginFormatAction]*plugin.FormattedContent,
	data plugindata.Data,
	publishAction *PluginPublishAction,
	requestedFormat *PluginFormatAction,
) (diags diagnostics.Diag) {
	log := appctx.Log(ctx)
	log = log.With("document", doc.GetTemplateName())

	docData := plugindata.Map{
		definitions.BlockKindContent: content.AsData(),
		definitions.BlockKindMeta:    doc.Meta(),
	}
	dataCtx := plugindata.Map{
		definitions.BlockKindData:     data,
		definitions.BlockKindDocument: docData,
	}

	format := publishAction.Format
	if requestedFormat != nil {
		format = requestedFormat
	}

	formattedContent, ok := formattedContents[format]
	if !ok {
		log.ErrorContext(
			ctx, "Formatted content for publish block not found",
			"publish_block", publishAction.FullBlockName(),
		)
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Formatted content not found for publish block",
			Detail: fmt.Sprintf(
				"Content from format `%s` not found for publish block `%s`",
				publishAction.Format.FullBlockName(),
				publishAction.FullBlockName(),
			),
		}}
	}

	log.DebugContext(
		ctx, "Executing publish block",
		"publish_block", publishAction.FullBlockName(),
	)

	documentName := doc.GetTemplateName()
	diag := publishAction.Publish(ctx, dataCtx, content, formattedContent, documentName)
	if diag != nil {
		diags.Extend(diag)
	}
	return diags
}

func LoadDocument(
	ctx context.Context,
	runners Runners,
	node *definitions.Document,
	dataCtx plugindata.Map,
) (_ *Document, diags diagnostics.Diag) {
	log := appctx.Log(ctx)

	doc := Document{
		Source:       node,
		meta:         node.Meta,
		Inputs:       node.Inputs,
		Vars:         node.Vars,
		RequiredVars: node.RequiredVars,
	}
	dataNames := make(map[[2]string]struct{})

	for _, child := range node.DataBlocks {
		dataAction, diag := LoadDataAction(ctx, runners, child, dataCtx)
		if diags.Extend(diag) {
			log.ErrorContext(
				ctx,
				"Error while loading data block",
				"err",
				diagnostics.GetDiagsDetails(diags),
			)
			return nil, diags
		}
		key := [2]string{dataAction.RunnerName, dataAction.BlockName}
		if _, found := dataNames[key]; found {
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Name conflict",
				Detail:   "Data block with the same name is already defined.",
				Subject:  &dataAction.SrcRange,
			})
		}
		dataNames[key] = struct{}{}
		doc.DataBlocks = append(doc.DataBlocks, dataAction)
	}

	if node.Title != nil {
		decoded, diag := LoadContent(ctx, runners, node.Title)
		if diags.Extend(diag) {
			log.ErrorContext(
				ctx,
				"Error while loading title",
				"err",
				diagnostics.GetDiagsDetails(diags),
			)
			return nil, diags
		}
		doc.Title = decoded
	}

	for _, child := range node.ContentTreeBlocks {
		decoded, diag := LoadContent(ctx, runners, child)
		if diags.Extend(diag) {
			log.ErrorContext(
				ctx, "Error while loading block",
				"err", diagnostics.GetDiagsDetails(diags),
			)
			return nil, diags
		}
		if utils.IsNil(decoded) {
			log.ErrorContext(
				ctx, "Received nil instead of decoded content tree block",
				"type", fmt.Sprintf("%T", decoded),
			)
			continue
		}
		doc.ContentTreeBlocks = append(doc.ContentTreeBlocks, decoded)
	}

	for _, child := range node.FormatBlocks {
		decoded, diag := LoadPluginFormatAction(ctx, runners, child, dataCtx)
		if diags.Extend(diag) {
			log.ErrorContext(
				ctx,
				"Error while loading `format` block",
				"err",
				diagnostics.GetDiagsDetails(diags),
			)
			return nil, diags
		}
		doc.FormatBlocks = append(doc.FormatBlocks, decoded)
	}

	for _, child := range node.PublishBlocks {

		decoded, diag := LoadPluginPublishAction(ctx, runners, runners, child, dataCtx)
		if diags.Extend(diag) {
			log.ErrorContext(
				ctx, "Error while loading publish block",
				"block_runner", child.RunnerName,
				"block_name", child.BlockName,
				"err", diagnostics.GetDiagsDetails(diags),
			)
			return nil, diags
		}
		if decoded == nil {
			for _, d := range diag {
				if d.Severity == hcl.DiagWarning {
					log.WarnContext(
						ctx, d.Summary,
						"block_runner", child.RunnerName,
						"block_name", child.BlockName,
						"diag", diag,
					)
				}
			}
			continue
		}
		doc.PublishBlocks = append(doc.PublishBlocks, decoded)
	}

	return &doc, diags
}
