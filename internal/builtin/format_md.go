package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/pelletier/go-toml/v2"
	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"gopkg.in/yaml.v3"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

func makeMarkdownFormatter(logger *slog.Logger, tracer trace.Tracer) *plugin.Formatter {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if tracer == nil {
		tracer = nooptrace.Tracer{}
	}
	return &plugin.Formatter{
		Doc:     "Formats content in Markdown",
		Format:  "md",
		FileExt: "md",
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "page_title",
					Doc:  "Markdown Page title",
					Type: cty.String,
				},
				{
					Name:        "frontmatter",
					Type:        plugindata.Encapsulated.CtyType(),
					Doc:         `Arbitrary key-value map to be put in the frontmatter`,
					Constraints: constraint.RequiredMeaningful,
					ExampleVal: cty.ObjectVal(map[string]cty.Value{
						"key": cty.StringVal("arbitrary value"),
						"key2": cty.MapVal(map[string]cty.Value{
							"nested_key": cty.NumberIntVal(42),
						}),
					}),
				},
				{
					Name:       "frontmatter_format",
					Type:       cty.String,
					Doc:        `Format of the frontmatter.`,
					DefaultVal: cty.StringVal("yaml"),
					OneOf: []cty.Value{
						cty.StringVal("yaml"),
						cty.StringVal("toml"),
						cty.StringVal("json"),
					},
				},
			},
		},
		FormatFunc: makeMarkdownFormatterFunc(logger, tracer),
	}
}

func makeMarkdownFormatterFunc(logger *slog.Logger, tracer trace.Tracer) plugin.FormatFunc {
	return func(ctx context.Context, params *plugin.FormatParams) (_ *plugin.FormattedContent, diags diagnostics.Diag) {
		document, _ := parseScope(params.DataContext)
		if document == nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse data context",
				Detail:   "document is not found",
			}}
		}
		// datactx := params.DataContext
		// datactx["format"] = plugindata.String(params.Format)

		frontmatterData, err := plugindata.Encapsulated.FromCty(params.Args.GetAttrVal("frontmatter"))

		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse frontmatter content",
				Detail:   err.Error(),
			}}
		}

		var frontmatterString *string
		var diag diagnostics.Diag

		if frontmatterData != nil {
			frontmatterMap, ok := (*frontmatterData).(plugindata.Map)
			if !ok {
				return nil, diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Failed to parse frontmatter content type",
					Detail: fmt.Sprintf(
						"Received invalid frontmatter data type `%T` while map is required",
						frontmatterData,
					),
				}}
			}
			format := params.Args.GetAttrVal("frontmatter_format").AsString()

			frontmatterString, diag = renderFrontmatter(format, frontmatterMap)
			if diags.Extend(diag) {
				return nil, diags
			}
		}

		logger.InfoContext(ctx, "Markdown FORMATTER CALLED", "params", params, "frontmatter", frontmatterString)

		content := fmt.Sprintf("MD CONTENT: %s\n", params.Content)

		return &plugin.FormattedContent{
			Content: []byte(content),
			Format:  params.Format,
		}, nil

		// var printer print.Printer
		// switch params.Format {
		// case plugin.OutputFormatMD:
		// 	printer = mdprint.New()
		// case plugin.OutputFormatHTML:
		// 	printer = htmlprint.New()
		// case plugin.OutputFormatPDF:
		// 	printer = pdfprint.New()
		// default:
		// 	return diagnostics.Diag{{
		// 		Severity: hcl.DiagError,
		// 		Summary:  "Unsupported format",
		// 		Detail:   "Only md, html and pdf formats are supported",
		// 	}}
		// }
		// printer = print.WithLogging(printer, logger, slog.String("format", params.Format.String()))
		// printer = print.WithTracing(printer, tracer, attribute.String("format", params.Format.String()))
		// pathAttr := params.Args.GetAttrVal("path")
		// if pathAttr.IsNull() || pathAttr.AsString() == "" {
		// 	return diagnostics.Diag{{
		// 		Severity: hcl.DiagError,
		// 		Summary:  "Failed to parse arguments",
		// 		Detail:   "path is required",
		// 	}}
		// }
		// path, err := templatePath(pathAttr.AsString(), datactx)
		// if err != nil {
		// 	return diagnostics.Diag{{
		// 		Severity: hcl.DiagError,
		// 		Summary:  "Failed to render a path value",
		// 		Detail:   err.Error(),
		// 	}}
		// }
		// logger.InfoContext(ctx, "Writing to a file", "path", path)
		// dir := filepath.Dir(path)
		// err = os.MkdirAll(dir, 0o755)
		// if err != nil {
		// 	return diagnostics.Diag{{
		// 		Severity: hcl.DiagError,
		// 		Summary:  "Failed to create a directory",
		// 		Detail:   err.Error(),
		// 	}}
		// }
		// fs, err := os.Create(path)
		// if err != nil {
		// 	return diagnostics.Diag{{
		// 		Severity: hcl.DiagError,
		// 		Summary:  "Failed to create a file",
		// 		Detail:   err.Error(),
		// 	}}
		// }
		// defer fs.Close()
		// err = printer.Print(ctx, fs, document)
		// if err != nil {
		// 	return diagnostics.Diag{{
		// 		Severity: hcl.DiagError,
		// 		Summary:  "Failed to write to a file",
		// 		Detail:   err.Error(),
		// 	}}
		// }
		// return nil
	}
}

func renderFrontmatter(format string, data plugindata.Map) (*string, diagnostics.Diag) {
	var result string
	var err error
	switch format {
	case "yaml":
		result, err = renderYAMLFrontMatter(data)
	case "toml":
		result, err = renderTOMLFrontMatter(data)
	case "json":
		result, err = renderJSONFrontMatter(data)
	default:
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Invalid frontmatter format",
			Detail:   fmt.Sprintf("Received unsupported frontmatter format `%s`", format),
		}}
	}
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to render frontmatter",
			Detail:   err.Error(),
		}}
	}
	return &result, nil
}

func renderYAMLFrontMatter(m plugindata.Map) (string, error) {
	var buf strings.Builder
	buf.WriteString("---\n")
	err := yaml.NewEncoder(&buf).Encode(m)
	if err != nil {
		return "", err
	}
	buf.WriteString("---")
	return buf.String(), nil
}

func renderTOMLFrontMatter(m plugindata.Map) (string, error) {
	var buf strings.Builder
	buf.WriteString("+++\n")
	err := toml.NewEncoder(&buf).Encode(m)
	if err != nil {
		return "", err
	}
	buf.WriteString("+++")
	return buf.String(), nil
}

func renderJSONFrontMatter(m plugindata.Map) (string, error) {
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	err := enc.Encode(m)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
