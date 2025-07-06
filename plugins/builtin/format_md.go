package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/pelletier/go-toml/v2"
	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"gopkg.in/yaml.v3"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

const (
	formatMarkdown = "md"
	joinBlocksWith = "\n\n"
)

func makeMarkdownFormatter(log *slog.Logger, tracer trace.Tracer) *plugin.Formatter {
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if tracer == nil {
		tracer = nooptrace.Tracer{}
	}
	return &plugin.Formatter{
		Doc:     "Formats content in Markdown",
		Format:  formatMarkdown,
		FileExt: "md",
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
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
		FormatFunc: makeMarkdownFormatterFunc(log, tracer),
	}
}

func renderHeadingEl(attrs plugindata.Map, nestingLevel int) string {
	body := string(attrs["body"].(plugindata.String))
	// size := attrsMap["size"].(plugindata.Number)

	size := nestingLevel + 1
	if size > 6 {
		size = 6
	}

	prefix := strings.Repeat("#", size)

	body = strings.TrimSpace(body)
	body = strings.ReplaceAll(body, "\n", " ")

	return fmt.Sprintf("%s %s", prefix, body)
}

func renderTextEl(attrs plugindata.Map) string {
	return string(attrs["body"].(plugindata.String))
}

func renderImageEl(attrs plugindata.Map) string {
	src := string(attrs["src"].(plugindata.String))
	src = strings.TrimSpace(src)
	src = strings.ReplaceAll(src, "\n", " ")

	alt := string(attrs["alt"].(plugindata.String))
	alt = strings.TrimSpace(alt)
	alt = strings.ReplaceAll(alt, "\n", " ")

	return fmt.Sprintf("![%s](%s)", alt, src)
}

func renderCodeEl(attrs plugindata.Map) string {
	body := string(attrs["body"].(plugindata.String))
	language := string(attrs["language"].(plugindata.String))

	return fmt.Sprintf("```%s\n%s\n```", language, body)
}

func renderListEl(attrs plugindata.Map) string {
	itemsRendered := attrs["items_rendered"].(plugindata.List)
	format := string(attrs["format"].(plugindata.String))

	prefix := "-"
	if format == "ordered" {
		prefix = "1."
	} else if format == "tasklist" {
		prefix = "- [ ]"
	}

	listValues := []string{}

	for _, val := range itemsRendered {
		valStr := string(val.(plugindata.String))

		listItemVal := fmt.Sprintf("%s %s", prefix, valStr)
		listValues = append(listValues, listItemVal)
	}

	return strings.Join(listValues, "\n")
}

func renderTableEl(attrs plugindata.Map) string {

	headers := attrs["items_rendered"].(plugindata.List)
	cellValues := string(attrs["format"].(plugindata.String))

	buf := &strings.Builder{}

	// https://github.com/olekukonko/tablewriter?tab=readme-ov-file#2-markdown-table

	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithRenderer(renderer.NewMarkdown()),
	)
	table.Header(headers)
	table.Bulk(cellValues)
	table.Render()

	// https://github.com/nao1215/markdown/blob/main/markdown.go#L348
	// 	table := tablewriter.NewWriter(buf)
	// 	table.SetNewLine("\n")
	// 	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	// 	table.SetCenterSeparator("|")
	// 	table.SetHeader(t.Header)

	return buf.String()
}

func renderElement(el plugin.Content, nestingLevel int) string {

	var attrs plugindata.Map

	if el.Kind() != plugin.SectionKind {
		data := el.AsData()
		attrsData, ok := data["attrs"]
		if !ok {
			return "<error-no-attrs>"
		}
		attrs, ok = attrsData.(plugindata.Map)
		if !ok {
			return "<error-unknown-type>"
		}
	}

	switch el.Kind() {
	case plugin.SectionKind:
		section := el.(*plugin.ContentSection)
		outputs := []string{}
		for _, child := range section.Children {
			outputs = append(outputs, renderElement(child, nestingLevel+1))
		}
		return strings.Join(outputs, joinBlocksWith)
	case plugin.EmptyKind:
		// nothing to add
	case plugin.HeadingKind:
		output := renderHeadingEl(attrs, nestingLevel)
		// MD022 - https://github.com/DavidAnson/markdownlint/blob/main/doc/md022.md
		// return fmt.Sprintf("\n%s\n", output)
		return output
	case plugin.TextKind:
		return renderTextEl(attrs)
	case plugin.ImageKind:
		return renderImageEl(attrs)
	case plugin.CodeKind:
		output := renderCodeEl(attrs)
		// MD031 - https://github.com/DavidAnson/markdownlint/blob/main/doc/md031.md
		// return fmt.Sprintf("\n%s\n", output)
		return output
	case plugin.ListKind:
		output := renderCodeEl(attrs)
		// MD032 - https://github.com/DavidAnson/markdownlint/blob/main/doc/md032.md
		//return fmt.Sprintf("\n%s\n", output)
		return output
	case plugin.TableKind:
		output := renderTableEl(attrs)
		// MD058 - https://github.com/DavidAnson/markdownlint/blob/main/doc/md058.md
		// return fmt.Sprintf("\n%s\n", output)
		return output
	case plugin.TOCKind:
		return renderCodeEl(attrs)
	}
	return "<error-unknown-element>"
}

func makeMarkdownFormatterFunc(log *slog.Logger, tracer trace.Tracer) plugin.FormatFunc {
	return func(ctx context.Context, params *plugin.FormatParams) (_ *plugin.FormattedContent, diags diagnostics.Diag) {

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

		outputs := []string{}

		for _, child := range section.Children {
			out := renderElement(child, 0)
			outputs = append(outputs, out)
		}

		var frontmatter *string

		frontmatterVal := params.Args.GetAttrVal("frontmatter")
		if !frontmatterVal.IsNull() {
			frontmatterData, err := plugindata.Encapsulated.FromCty(frontmatterVal)
			if err != nil {
				return nil, diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Failed to parse frontmatter content",
					Detail:   err.Error(),
				}}
			}

			if frontmatterData != nil {
				frontmatterMap, ok := (*frontmatterData).(plugindata.Map)
				if !ok {
					return nil, diagnostics.Diag{{
						Severity: hcl.DiagError,
						Summary:  "Failed to parse frontmatter content type",
						Detail: fmt.Sprintf(
							"Received invalid frontmatter data type `%T` while map is expected",
							frontmatterData,
						),
					}}
				}
				format := params.Args.GetAttrVal("frontmatter_format").AsString()

				var diag diagnostics.Diag
				frontmatter, diag = renderFrontmatter(format, frontmatterMap)
				if diags.Extend(diag) {
					return nil, diags
				}
			}
		}

		if frontmatter != nil {
			// Prepend frontmatter to the output
			outputs = append([]string{*frontmatter}, outputs...)
		}

		content := strings.Join(outputs, joinBlocksWith)
		content = strings.TrimSpace(content)

		if content != "" {
			content += "\n"
		}

		return &plugin.FormattedContent{
			Content: []byte(content),
			Format:  formatMarkdown,
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
