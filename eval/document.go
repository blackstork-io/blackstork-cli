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

	DefaultPublish *PluginPublishAction
	DefaultFormat  *PluginFormatAction
}

func (doc *Document) FetchData(ctx context.Context) (plugindata.Map, diagnostics.Diag) {
	return executeDataBlocksAsync(ctx, doc, nil)
}

func (doc *Document) FetchDataWithPath(ctx context.Context, path []string) (plugindata.Data, diagnostics.Diag) {
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
) (formattedContentMap map[string]*plugin.FormattedContent, diags diagnostics.Diag) {
	log := appctx.Log(ctx)

	// Collecting all required formats from publish blocks
	formatters := map[string]*PluginFormatAction{}

	for _, block := range doc.PublishBlocks {

		if block.Format == nil {
			continue
		}

		format := *block.Format
		if _, ok := formatters[format]; ok {
			// already registered
			continue
		}

		// FIXME: we randomly pick any format from supported, which is not good
		// there should be a way to specify exact formatter name in a publish block
		var formatBlock *PluginFormatAction
		for _, fb := range doc.FormatBlocks {
			if fb.RunnerName == format {
				formatBlock = fb
				break
			}
		}

		if formatBlock == nil && format == doc.DefaultFormat.Formatter.Format {
			formatBlock = doc.DefaultFormat
			doc.FormatBlocks = append(doc.FormatBlocks, formatBlock)
			log.InfoContext(ctx, "Registering default formatter", "formatter", formatBlock.RunnerName)
		}

		if formatBlock == nil {
			diags.Extend(diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Formatter required by publisher not found",
				Detail: fmt.Sprintf(
					"Definition block for `%s` format required by `%s` publisher not found",
					format,
					block.RunnerName,
				),
			}})
			continue
		}

		formatters[format] = formatBlock
	}

	if diags.HasErrors() {
		return formattedContentMap, diags
	}

	contentData := content.AsData()

	// prepare data for the formatter
	docData := plugindata.Map{
		definitions.BlockKindContent: contentData,
		definitions.BlockKindMeta:    doc.Meta(),
	}
	data[definitions.BlockKindDocument] = docData

	formattedContentMap = map[string]*plugin.FormattedContent{}
	for format, formatter := range formatters {
		formattedContent, diag := formatter.Execute(ctx, data, contentData)
		if diags.Extend(diag) {
			// do not return but collect all errors
			continue
		}
		formattedContentMap[format] = formattedContent
	}

	return formattedContentMap, diags
}

func (doc *Document) Publish(
	ctx context.Context,
	content plugin.Content,
	formattedContentMap map[string]*plugin.FormattedContent,
	data plugindata.Data,
) diagnostics.Diag {
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

	var diags diagnostics.Diag

	if len(doc.PublishBlocks) == 0 {
		log.WarnContext(ctx, "No publish blocks found")
		return diags
	}

	// Calling publish blocks while passing in a requested formatter

	for _, block := range doc.PublishBlocks {

		var formattedContent *plugin.FormattedContent
		if block.Format != nil {
			var ok bool
			formattedContent, ok = formattedContentMap[*block.Format]
			if !ok {
				log.ErrorContext(ctx, "Formatted content not available for publisher")
				return diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Formatted content not found while publishing content",
					Detail: fmt.Sprintf(
						"Content formatted as `%s` is not availalbe for publisher `%s.%s`",
						*block.Format,
						block.RunnerName,
						block.BlockName,
					),
				}}
			}
		}

		log.DebugContext(
			ctx, "Executing publish block",
			"publisher", block.RunnerName,
			"block_name", block.BlockName,
		)

		documentName := doc.GetTemplateName()
		diag := block.Publish(ctx, dataCtx, content, formattedContent, documentName)
		if diag != nil {
			diags.Extend(diag)
		}
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
			log.ErrorContext(ctx, "Error while loading data block", "err", diagnostics.GetDiagsDetails(diags))
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
			log.ErrorContext(ctx, "Error while loading title", "err", diagnostics.GetDiagsDetails(diags))
			return nil, diags
		}
		doc.Title = decoded
	}

	for _, child := range node.ContentTreeBlocks {
		decoded, diag := LoadContent(ctx, runners, child)
		if diags.Extend(diag) {
			log.ErrorContext(ctx, "Error while loading block", "err", diagnostics.GetDiagsDetails(diags))
			return nil, diags
		}
		if utils.IsNil(decoded) {
			log.ErrorContext(ctx, "Received nil instead of decoded content tree block", "type", fmt.Sprintf("%T", decoded))
			continue
		}
		doc.ContentTreeBlocks = append(doc.ContentTreeBlocks, decoded)
	}

	for _, child := range node.FormatBlocks {
		decoded, diag := LoadPluginFormatAction(ctx, runners, child, dataCtx)
		if diags.Extend(diag) {
			log.ErrorContext(ctx, "Error while loading format block", "err", diagnostics.GetDiagsDetails(diags))
			return nil, diags
		}
		doc.FormatBlocks = append(doc.FormatBlocks, decoded)
	}

	for _, child := range node.PublishBlocks {
		decoded, diag := LoadPluginPublishAction(ctx, runners, child, dataCtx)
		if diags.Extend(diag) {
			log.ErrorContext(ctx, "Error while loading publish block", "err", diagnostics.GetDiagsDetails(diags))
			return nil, diags
		}
		if decoded == nil {
			for _, d := range diag {
				if d.Severity == hcl.DiagWarning {
					log.WarnContext(ctx, d.Summary, "block_runner", child.RunnerName, "block_name", child.BlockName, "diag", diag)
				}
			}
			continue
		}
		doc.PublishBlocks = append(doc.PublishBlocks, decoded)
	}

	// Default publisher / formatter instances
	var diag diagnostics.Diag

	doc.DefaultPublish, diag = LoadStdoutPluginPublishAction(ctx, runners, "md")
	if diags.Extend(diag) {
		return nil, diags
	}

	doc.DefaultFormat, diag = LoadMarkdownPluginFormatAction(ctx, runners)
	if diags.Extend(diag) {
		return nil, diags
	}

	return &doc, diags
}
