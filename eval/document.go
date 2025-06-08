package eval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

type Document struct {
	Source       *definitions.Document
	Meta         *definitions.MetaBlock
	Vars         *definitions.Vars
	RequiredVars []string
	DataBlocks   []*PluginDataAction

	ContentTreeBlocks []ContentTreeEvalBlock

	PublishBlocks []*PluginPublishAction
	FormatBlocks  []*PluginFormatAction

	DefaultPublish *PluginPublishAction
	DefaultFormat  *PluginFormatAction
}

func (doc *Document) FetchData(ctx context.Context) (plugindata.Data, diagnostics.Diag) {
	return executeDataBlocksAsync(ctx, doc, nil)
}

func (doc *Document) FetchDataWithPath(ctx context.Context, path []string) (plugindata.Data, diagnostics.Diag) {
	return executeDataBlocksAsync(ctx, doc, path)
}

func (doc *Document) GetTemplateName() string {
	return doc.Source.Source.Name
}

func (doc *Document) RenderContent(
	ctx context.Context,
	docDataCtx plugindata.Map,
	requiredTags []string,
) (*plugin.ContentSection, plugindata.Map, diagnostics.Diag) {
	log := fabctx.GetLog(ctx)
	log.InfoContext(ctx, "Processing a document template", "document", doc.Source.Source.Name)

	data, diags := doc.FetchData(ctx)
	if diags.HasErrors() {
		return nil, nil, diags
	}

	docDataCtx[definitions.BlockKindData] = data

	diag := applyBlockDataToDataCtx(
		ctx,
		doc.Vars,
		doc.RequiredVars,
		doc.Meta,
		docDataCtx,
	)
	if diags.Extend(diag) {
		return nil, nil, diags
	}

	result, diag := executeContentBlocksAsync(ctx, doc, requiredTags, docDataCtx)
	if diags.Extend(diag) {
		return nil, nil, diags
	}

	return result, docDataCtx, diags
}

func (doc *Document) FormatContent(
	ctx context.Context,
	content *plugin.ContentSection,
	data plugindata.Map,
) (formattedContentMap map[string]*plugin.FormattedContent, diags diagnostics.Diag) {
	log := fabctx.GetLog(ctx)

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

		var formatBlock *PluginFormatAction
		for _, fb := range doc.FormatBlocks {
			if fb.BlockRunnerName == format {
				formatBlock = fb
				break
			}
		}

		if formatBlock == nil && format == doc.DefaultFormat.Formatter.Format {
			formatBlock = doc.DefaultFormat
			doc.FormatBlocks = append(doc.FormatBlocks, formatBlock)
			log.InfoContext(ctx, "Registering a default formatter", "formatter", formatBlock.BlockRunnerName)
		}

		if formatBlock == nil {
			diags.Extend(diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Formatter required by publisher not found",
				Detail: fmt.Sprintf(
					"Definition block for `%s` format required by `%s` publisher not found",
					format,
					block.BlockRunnerName,
				),
			}})
			continue
		}

		formatters[format] = formatBlock
	}

	if diags.HasErrors() {
		return
	}

	// prepare data for the formatter
	docData := plugindata.Map{
		definitions.BlockKindContent: content.AsData(),
	}
	if doc.Meta != nil {
		docData[definitions.BlockKindMeta] = doc.Meta.AsPluginData()
	}
	dataCtx := plugindata.Map{
		definitions.BlockKindData:     data,
		definitions.BlockKindDocument: docData,
	}

	formattedContentMap = map[string]*plugin.FormattedContent{}
	for format, formatter := range formatters {
		formattedContent, diag := formatter.Execute(ctx, dataCtx, content.AsData())
		if diags.Extend(diag) {
			// do not return but collect all errors
			continue
		}
		formattedContentMap[format] = formattedContent
	}

	return
}

func (doc *Document) Publish(
	ctx context.Context,
	content plugin.Content,
	formattedContentMap map[string]*plugin.FormattedContent,
	data plugindata.Data,
) diagnostics.Diag {
	log := fabctx.GetLog(ctx)
	log = log.With("document", doc.GetTemplateName())

	docData := plugindata.Map{
		definitions.BlockKindContent: content.AsData(),
	}
	if doc.Meta != nil {
		docData[definitions.BlockKindMeta] = doc.Meta.AsPluginData()
	}
	dataCtx := plugindata.Map{
		definitions.BlockKindData:     data,
		definitions.BlockKindDocument: docData,
	}

	// Calling publish blocks while passing in a requested formatter

	var diags diagnostics.Diag
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
						block.BlockRunnerName,
						block.BlockName,
					),
				}}
			}
		}

		log.DebugContext(
			ctx, "Executing a publish block",
			"publisher", block.BlockRunnerName,
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
	plugins Plugins,
	node *definitions.Document,
) (_ *Document, diags diagnostics.Diag) {
	doc := Document{
		Source:       node,
		Meta:         node.Meta,
		Vars:         node.Vars,
		RequiredVars: node.RequiredVars,
	}
	dataNames := make(map[[2]string]struct{})

	for _, child := range node.DataBlocks {
		dataAction, diag := LoadDataAction(ctx, plugins, child)
		if diags.Extend(diag) {
			return nil, diags
		}
		key := [2]string{dataAction.BlockRunnerName, dataAction.BlockName}
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

	for _, child := range node.ContentTreeBlocks {
		decoded, diag := LoadContent(ctx, plugins, child)
		if diags.Extend(diag) {
			return nil, diags
		}
		doc.ContentTreeBlocks = append(doc.ContentTreeBlocks, decoded)
	}

	for _, child := range node.FormatBlocks {
		decoded, diag := LoadPluginFormatAction(ctx, plugins, child)
		if diags.Extend(diag) {
			return nil, diags
		}
		doc.FormatBlocks = append(doc.FormatBlocks, decoded)
	}

	for _, child := range node.PublishBlocks {
		decoded, diag := LoadPluginPublishAction(ctx, plugins, child)
		if diags.Extend(diag) {
			return nil, diags
		}
		doc.PublishBlocks = append(doc.PublishBlocks, decoded)
	}

	// Default publisher / formatter instances
	var diag diagnostics.Diag

	doc.DefaultPublish, diag = LoadStdoutPluginPublishAction(ctx, plugins, "md")
	if diags.Extend(diag) {
		return nil, diags
	}

	doc.DefaultFormat, diag = LoadMarkdownPluginFormatAction(ctx, plugins)
	if diags.Extend(diag) {
		return nil, diags
	}

	return &doc, diags
}
