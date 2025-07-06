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

	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/utils"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

//go:embed format_html_document.gotempl
var documentTemplStr string
var docTempl = template.Must(template.New("document").Parse(documentTemplStr))

type TypedBlock struct {
	kind   string
	runner string
}

type NamedBlock struct {
	kind   string
	runner string
	name   string
}

type Data struct {
	Title       string
	Description string
	Content     template.HTML
	CSS         template.CSS
	JS          template.JS
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
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "page_title",
					Doc:  "HTML page title",
					Type: cty.String,
				},
				{
					Name: "page_description",
					Doc:  "HTML page description",
					Type: cty.String,
				},
				{
					Name:        "template_per_type",
					Doc:         "HTML templates for document, section, and content blocks. For content blocks, the key must include the content provider.",
					Type:        plugindata.Encapsulated.CtyType(),
					Constraints: constraint.NonEmpty,
					ExampleVal: cty.ObjectVal(map[string]cty.Value{
						"content.text": cty.StringVal(`<span class="text-block">{{ .self.value }}</span>`),
						"content.image": cty.StringVal(
							`<img src="{{ .self.src }}" alt="{{ .self.alt }}" class="img-w-10" />`,
						),
					}),
				},
				{
					Name:        "template_per_block",
					Doc:         "HTML templates for specific named blocks.",
					Type:        plugindata.Encapsulated.CtyType(),
					Constraints: constraint.NonEmpty,
					ExampleVal: cty.ObjectVal(map[string]cty.Value{
						"content.text.foo": cty.StringVal(`<span class="text-block">{{ .self.value }}</span>`),
						"section.bar":      cty.StringVal(`<h1>{{ .self.title.value }}</h1><p>{{ .self.content }}</p>`),
					}),
				},
				{
					Name: "css_inline",
					Doc:  "CSS code to include inline",
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
		dataCtx["format"] = plugindata.String(params.Format)

		section, err := parseContentSection(params.Content)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse document content",
				Detail:   fmt.Sprintf("Error while parsing document content: %s", err),
			}}
		}

		// collect templates per typed block, for example for:
		// `content.text`
		// `content.list`
		templatePerTypedBlock, diag := parseTmplPerTypeMap(params.Args.GetAttrVal("templates_per_type"))
		if diag.HasErrors() {
			return nil, diag
		}

		// collect templates per named block, for example for:
		// `content.text.foo`
		templatePerNamedBlock, diag := parseTmplPerBlockMap(params.Args.GetAttrVal("templates_per_block"))
		if diag.HasErrors() {
			return nil, diag
		}

		log.DebugContext(ctx, "Decoded arguments", "per_type", templatePerTypedBlock, "per_name", templatePerNamedBlock)

		outputs := renderRecursively(section, templatePerTypedBlock, templatePerNamedBlock)

		// Deduce page title
		pageTitle := ""
		pageTitleVal := params.Args.GetAttrVal("page_title")
		if !pageTitleVal.IsNull() {
			log.WarnContext(ctx, "WHERE THE VARS", "ctx", params.DataContext["vars"])
			for k, v := range params.DataContext {
				fmt.Printf(">>> %s:\t%s\n", k, v)
			}
			pageTitle, err = renderText(
				pageTitleVal.AsString(),
				params.DataContext,
			)
			if err != nil {
				return nil, diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Failed to render the title value as a template",
					Detail:   err.Error(),
				}}
			}
		} else {
			firstTitleElement := firstTitle(section)
			if firstTitleElement != nil {
				titleData := firstTitleElement.Attr("body")
				if titleData != nil {
					pageTitle = string(titleData.(plugindata.String))
				}
			}
		}

		// Deduce description title
		var pageDescription string
		pageDescriptionVal := params.Args.GetAttrVal("page_description")
		if !pageDescriptionVal.IsNull() {
			pageDescription = pageDescriptionVal.AsString()
		}

		log.InfoContext(ctx, "Document title figured out", "title", pageTitle, "description", pageDescription)

		output := strings.Join(outputs, "\n\n")

		data := Data{
			Title:       "Untitled",
			Content:     template.HTML(output),
			Description: pageDescription,
			CSS:         template.CSS(params.Args.GetAttrVal("css_inline").AsString()),
			JS:          template.JS(params.Args.GetAttrVal("js_inline").AsString()),
			JSSources: utils.FnMap(
				params.Args.GetAttrVal("js_sources").AsValueSlice(),
				func(v cty.Value) template.URL {
					return template.URL(v.AsString())
				},
			),
			CSSSources: utils.FnMap(
				params.Args.GetAttrVal("js_sources").AsValueSlice(),
				func(v cty.Value) template.URL {
					return template.URL(v.AsString())
				},
			),
		}

		var resultBuff bytes.Buffer
		err = docTempl.Execute(&resultBuff, data)
		if err != nil {
			log.ErrorContext(ctx, "Error rendering a page", "err", err)
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Error rendering HTML page",
				Detail:   fmt.Sprintf("Error while rendering HTML page: %s", err),
			}}
		}

		return &plugin.FormattedContent{
			Content: resultBuff.Bytes(),
			Format:  params.Format,
		}, nil

	}
}

func parseTmplPerTypeMap(attrVal cty.Value) (result map[TypedBlock]string, diags diagnostics.Diag) {
	result = map[TypedBlock]string{}

	if attrVal.IsNull() {
		return result, diags
	}

	tmplPerTypeData, err := plugindata.Encapsulated.FromCty(attrVal)
	if err != nil {
		return result, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse `template_per_type` value",
			Detail:   err.Error(),
		}}
	}

	if tmplPerTypeData == nil {
		return result, diags
	}

	tmplPerTypeMap, ok := (*tmplPerTypeData).(plugindata.Map)
	if !ok {
		return result, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse `template_per_type` value type",
			Detail: fmt.Sprintf(
				"Received invalid data type `%T` while map is expected",
				tmplPerTypeData,
			),
		}}
	}

	for key, val := range tmplPerTypeMap {

		valStr, ok := val.(plugindata.String)
		if !ok {
			return result, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary: fmt.Sprintf(
					"Failed to parse template value for key `%s` in `template_per_type` data",
					key,
				),
				Detail: fmt.Sprintf(
					"Received invalid data type `%T` while string value is expected",
					val,
				),
			}}
		}

		k, err := parseBlockType(key)
		if err != nil {
			return result, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary: fmt.Sprintf(
					"Failed to parse key `%s` in `template_per_type` data",
					key,
				),
				Detail: err.Error(),
			}}
		}

		result[*k] = string(valStr)
	}
	return result, diags
}

func parseTmplPerBlockMap(attrVal cty.Value) (result map[NamedBlock]string, diags diagnostics.Diag) {
	result = map[NamedBlock]string{}

	if attrVal.IsNull() {
		return result, diags
	}

	tmplPerTypeData, err := plugindata.Encapsulated.FromCty(attrVal)
	if err != nil {
		return result, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse `template_per_block` value",
			Detail:   err.Error(),
		}}
	}

	if tmplPerTypeData == nil {
		return result, diags
	}

	tmplPerTypeMap, ok := (*tmplPerTypeData).(plugindata.Map)
	if !ok {
		return result, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse `template_per_block` value type",
			Detail: fmt.Sprintf(
				"Received invalid data type `%T` while map is expected",
				tmplPerTypeData,
			),
		}}
	}

	for key, val := range tmplPerTypeMap {
		valStr, ok := val.(plugindata.String)
		if !ok {
			return result, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary: fmt.Sprintf(
					"Failed to parse template value for key `%s` in `template_per_block` data",
					key,
				),
				Detail: fmt.Sprintf(
					"Received invalid data type `%T` while string value is expected",
					val,
				),
			}}
		}
		k, err := parseBlockName(key)
		if err != nil {
			return result, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary: fmt.Sprintf(
					"Failed to parse key `%s` in `template_per_block` data",
					key,
				),
				Detail: err.Error(),
			}}
		}

		result[*k] = string(valStr)
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
			return nil, fmt.Errorf("invalid block type found in block name `%s`", val)
		}
	} else if len(parts) == 2 {
		kind = parts[0]

		if kind != definitions.BlockKindContent {
			return nil, fmt.Errorf("invalid block type found in block name `%s`", val)
		}
		runner = parts[1]
	} else {
		return nil, fmt.Errorf("error parsing block name `%s`", val)
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

func renderRecursively(
	el plugin.Content,
	templatePerType map[TypedBlock]string,
	templatePerName map[NamedBlock]string,
) []string {
	outputs := []string{}

	switch el := el.(type) {
	case *plugin.ContentSection:
		slog.Debug("In section", "self", el.Self(), "meta", el.Meta())
		for _, c := range el.Children {
			childOutputs := renderRecursively(c, templatePerType, templatePerName)
			outputs = append(outputs, childOutputs...)
		}

		// render section after children

	case *plugin.ContentElement:
		slog.Debug("In element", "self", el.Self(), "meta", el.Meta())
		output := "what"
		outputs = append(outputs, output)
	}
	return outputs
}
