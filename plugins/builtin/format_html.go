// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package builtin

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

//go:embed format_html_document.gotempl
var documentTemplStr string

type TypedBlock struct {
	kind   string
	runner string
}

type NamedBlock struct {
	kind   string
	runner string
	name   string
}

type HTMLData struct {
	Title       string
	Description string
	Content     template.HTML
	CSS         *template.CSS
	TailwindCSS *template.CSS
	JS          *template.JS
	JSSources   []template.URL
	CSSSources  []template.URL
}

func makeHTMLFormatter(logger *slog.Logger, tracer trace.Tracer) *plugin.Formatter {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if tracer == nil {
		tracer = nooptrace.Tracer{}
	}
	return &plugin.Formatter{
		Doc:     "Formats content in HTML",
		Format:  "html",
		FileExt: "html",
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "page_template",
					Type:        cty.String,
					Constraints: constraint.RequiredMeaningful,
					DefaultVal:  cty.StringVal(documentTemplStr),
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "template_per_type",
					Doc:  "HTML templates for document, section, and content blocks. For content blocks, the key must include the content provider.",
					Type: cty.Map(cty.DynamicPseudoType),
					ExampleVal: cty.ObjectVal(map[string]cty.Value{
						"content.text": cty.StringVal(`<span class="text-block">{{ .value }}</span>`),
						"content.image": cty.StringVal(
							`<img src="{{ .src }}" alt="{{ .alt }}" class="img-w-10" />`,
						),
					}),
				},
				{
					Name: "template_per_block",
					Doc:  "HTML templates for specific named blocks.",
					Type: cty.Map(cty.DynamicPseudoType),
					ExampleVal: cty.ObjectVal(map[string]cty.Value{
						"content.text.foo": cty.StringVal(`<span class="text-block">{{ .block.value }}</span>`),
						"section.bar": cty.StringVal(
							`<h1>{{ .title }}</h1><p>{{ .content }}</p>`,
						),
					}),
				},
				{
					Name: "css_inline",
					Doc:  "CSS code to include inline",
					Type: cty.String,
				},
				{
					Name: "css_inline_tailwind",
					Doc:  "CSS code to include inline inside <style> tag of type `text/tailwindcss`",
					Type: cty.String,
				},
				{
					Name: "js_inline",
					Doc:  "JS code to include inline",
					Type: cty.String,
				},
				{
					Name: "css_sources",
					Doc:  "CSS source URLs",
					Type: cty.List(cty.String),
				},
				{
					Name: "js_sources",
					Doc:  "JS source URLs",
					Type: cty.List(cty.String),
				},
			},
		},
		FormatFunc: makeHTMLFormatterFunc(logger, tracer),
	}
}

func makeHTMLFormatterFunc(log *slog.Logger, tracer trace.Tracer) plugin.FormatFunc {
	return func(ctx context.Context, params *plugin.FormatParams) (*plugin.FormattedContent, diagnostics.Diag) {
		dataCtx := params.DataContext
		if dataCtx == nil {
			dataCtx = plugindata.Map{}
		}
		dataCtx["format"] = plugindata.String(params.Format)

		section, err := ParseContentSection(params.Content)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse document content",
				Detail:   fmt.Sprintf("Error while parsing document content: %s", err),
			}}
		}

		pageTemplate := params.Config.GetAttrVal("page_template").AsString()
		if pageTemplate == "" {
			log.ErrorContext(ctx, "Non-empty page template must be provided")
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Subject:  &params.Config.ContentsRange,
				Summary:  "Received empty page template",
				Detail:   "Non-empty template must be provided in configuration of HTML formatter",
			}}
		}

		docTempl, err := template.New("document").Parse(pageTemplate)
		if err != nil {
			log.ErrorContext(ctx, "Error while parsing document template", "err", err)
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Subject:  &params.Config.ContentsRange,
				Summary:  "Error while parsing document template",
				Detail:   fmt.Sprintf("%s", err),
			}}
		}

		// collect templates per typed block, for example for:
		// `content.text`
		// `content.list`
		templatePerTypedBlock, diag := ParseTmplPerTypeMap(params.Args.GetAttrVal("template_per_type"))
		if diag.HasErrors() {
			log.ErrorContext(
				ctx, "Error while parsing `template_per_type` value",
				"err", diag,
				"val", params.Args.GetAttrVal("template_per_type"),
				"val_type", fmt.Sprintf("%T", params.Args.GetAttrVal("template_per_type")),
			)
			return nil, diag
		}

		// collect templates per named block, for example for:
		// `content.text.foo`
		templatePerNamedBlock, diag := ParseTmplPerBlockMap(params.Args.GetAttrVal("template_per_block"))
		if diag.HasErrors() {
			return nil, diag
		}

		log.DebugContext(
			ctx, "Decoded arguments",
			"per_type_count", len(templatePerTypedBlock),
			"per_name_count", len(templatePerNamedBlock),
		)

		// HTML page details

		// Deduce page title
		pageTitle := "Untitled"

		if foundTitle := FirstTitleValue(section); foundTitle != nil {
			pageTitle = *foundTitle
		}

		log.DebugContext(ctx, "Document title figured out", "title", pageTitle)

		var css *template.CSS
		cssInlineVal := params.Args.GetAttrVal("css_inline")
		if cssInlineVal != cty.NilVal {
			cssObj := template.CSS(cssInlineVal.AsString()) //nolint:gosec // Explicit raw CSS is the purpose of this trusted template option.
			css = &cssObj
		}

		var tailwindCSS *template.CSS
		cssInlineTailwindVal := params.Args.GetAttrVal("css_inline_tailwind")
		if cssInlineTailwindVal != cty.NilVal {
			cssObj := template.CSS(cssInlineTailwindVal.AsString()) //nolint:gosec // Explicit raw CSS is the purpose of this trusted template option.
			tailwindCSS = &cssObj
		}

		var js *template.JS
		jsInlineVal := params.Args.GetAttrVal("js_inline")
		if jsInlineVal != cty.NilVal {
			jsObj := template.JS(jsInlineVal.AsString()) //nolint:gosec // Explicit raw JavaScript is the purpose of this trusted template option.
			js = &jsObj
		}

		var jsSources []cty.Value
		jsSourcesVal := params.Args.GetAttrVal("js_sources")
		if jsSourcesVal != cty.NilVal {
			jsSources = jsSourcesVal.AsValueSlice()
		}

		var cssSources []cty.Value
		cssSourcesVal := params.Args.GetAttrVal("css_sources")
		if cssSourcesVal != cty.NilVal {
			cssSources = cssSourcesVal.AsValueSlice()
		}

		// Rendering page content
		component := ContentHTML(
			ctx,
			log,
			templatePerTypedBlock,
			templatePerNamedBlock,
			section,
			DocumentRootLevel,
			nil,
		)

		buf := new(bytes.Buffer)
		if err = component.Render(ctx, buf); err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to render document content",
				Detail:   fmt.Sprintf("Error while rendering document content: %s", err),
			}}
		}

		output := buf.String()

		data := HTMLData{
			Title:       pageTitle,
			Content:     template.HTML(output), //nolint:gosec // output is rendered document HTML and must remain markup.
			Description: "",
			CSS:         css,
			TailwindCSS: tailwindCSS,
			JS:          js,
			JSSources: utils.FnMap(
				jsSources,
				func(v cty.Value) template.URL {
					return template.URL(v.AsString()) //nolint:gosec // Explicit stylesheet URLs are trusted template configuration.
				},
			),
			CSSSources: utils.FnMap(
				cssSources,
				func(v cty.Value) template.URL {
					return template.URL(v.AsString()) //nolint:gosec // Explicit script URLs are trusted template configuration.
				},
			),
		}

		var resultBuff bytes.Buffer
		err = docTempl.Execute(&resultBuff, data)
		if err != nil {
			log.ErrorContext(ctx, "Error rendering HTML page", "err", err)
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Error rendering HTML page",
				Detail:   fmt.Sprintf("%s", err),
			}}
		}

		return &plugin.FormattedContent{
			Content: resultBuff.Bytes(),
			Format:  params.Format,
		}, nil
	}
}

func ParseTmplPerTypeMap(attrVal cty.Value) (result map[TypedBlock]string, diags diagnostics.Diag) {
	result = map[TypedBlock]string{}

	if attrVal.IsNull() {
		return result, diags
	}

	for typeKey, ctyVal := range attrVal.AsValueMap() {

		k, err := parseBlockType(typeKey)
		if err != nil {
			return result, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary: fmt.Sprintf(
					"Failed to parse key `%s` in `template_per_type` map",
					typeKey,
				),
				Detail: err.Error(),
			}}
		}

		value := ctyVal.AsString()

		result[*k] = value
	}

	return result, diags
}

func ParseTmplPerBlockMap(attrVal cty.Value) (result map[NamedBlock]string, diags diagnostics.Diag) {
	result = map[NamedBlock]string{}

	if attrVal.IsNull() {
		return result, diags
	}

	for nameKey, ctyVal := range attrVal.AsValueMap() {

		k, err := parseBlockName(nameKey)
		if err != nil {
			return result, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary: fmt.Sprintf(
					"Failed to parse key `%s` in `template_per_type` map",
					nameKey,
				),
				Detail: err.Error(),
			}}
		}

		value := ctyVal.AsString()

		result[*k] = value
	}

	return result, diags
}

func parseBlockType(val string) (*TypedBlock, error) {
	// valid options:
	// - document
	// - section
	// - content.bar

	parts := strings.SplitN(val, ".", 2)
	var kind string
	var runner string
	if len(parts) == 1 {
		kind = parts[0]
		if kind != definitions.BlockKindSection && kind != definitions.BlockKindDocument {
			return nil, fmt.Errorf("invalid block type found: `%s`", val)
		}
	} else if len(parts) == 2 {
		kind = parts[0]

		if kind != definitions.BlockKindContent {
			return nil, fmt.Errorf("invalid block type found: `%s`", val)
		}
		runner = parts[1]
	} else {
		return nil, fmt.Errorf("error parsing block type `%s`", val)
	}

	return &TypedBlock{kind, runner}, nil
}

func parseBlockName(val string) (*NamedBlock, error) {
	// valid options:
	// - document.foo
	// - section.foo
	// - content.bar.baz

	parts := strings.SplitN(val, ".", 3)
	var kind string
	var runner string
	var name string
	if len(parts) == 2 {
		kind = parts[0]
		if kind != definitions.BlockKindSection && kind != definitions.BlockKindDocument {
			return nil, fmt.Errorf("invalid block type found in block name `%s`", val)
		}
		name = parts[1]
	} else if len(parts) == 3 {
		kind = parts[0]

		if kind != definitions.BlockKindContent {
			return nil, fmt.Errorf("invalid block type found in block name `%s`", val)
		}

		runner = parts[1]
		name = parts[2]
	} else {
		return nil, fmt.Errorf("error parsing block name `%s`", val)
	}

	return &NamedBlock{kind, runner, name}, nil
}
