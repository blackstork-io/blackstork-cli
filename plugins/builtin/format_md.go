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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/pelletier/go-toml/v2"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"gopkg.in/yaml.v3"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
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
					Name: "frontmatter",
					Type: plugindata.Encapsulated.CtyType(),
					Doc:  `Arbitrary key-value map to be put in the frontmatter`,
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

func renderHeadingEl(attrs plugindata.Map) string {
	value := string(attrs["value"].(plugindata.String))
	size := int(attrs["size"].(plugindata.Number))
	prefix := strings.Repeat("#", size)

	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\n", " ")

	// return fmt.Sprintf("%s %s", prefix, value)
	return fmt.Sprintf("%s %s", prefix, value)
}

func renderTextEl(attrs plugindata.Map) string {
	return string(attrs["value"].(plugindata.String))
}

func renderQuoteEl(attrs plugindata.Map) string {
	value := string(attrs["value"].(plugindata.String))
	newLines := []string{}
	for line := range strings.Lines(value) {
		newLines = append(newLines, "> "+line)
	}
	return strings.Join(newLines, "\n")
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
	value := string(attrs["value"].(plugindata.String))
	language := string(attrs["language"].(plugindata.String))

	return fmt.Sprintf("```%s\n%s\n```", language, value)
}

func renderTOCEl(attrs plugindata.Map) string {
	headings := attrs["headings"].(plugindata.List)

	if len(headings) == 0 {
		return ""
	}

	isOrdered := attrs["is_ordered"].(plugindata.Bool)

	prefix := "-"
	if bool(isOrdered) {
		prefix = "1."
	}

	tocLines := []string{}

	sharedLevel := math.MaxInt32
	for _, headAttrs := range headings {
		headAttrsMap := headAttrs.(plugindata.Map)
		level := headAttrsMap["level"].(plugindata.Number)

		sharedLevel = int(math.Min(float64(sharedLevel), float64(level)))
	}

	for _, headAttrs := range headings {

		headAttrsMap := headAttrs.(plugindata.Map)
		value := headAttrsMap["value"].(plugindata.String)
		level := headAttrsMap["level"].(plugindata.Number)

		lineLevel := math.Max(0, float64(int(level)-sharedLevel))
		// 3 spaces for "1. "
		preprefix := strings.Repeat("   ", int(lineLevel))

		itemPrefix := preprefix + prefix
		line := fmt.Sprintf("%s %s", itemPrefix, value)

		tocLines = append(tocLines, line)
	}

	return strings.Join(tocLines, "\n")
}

func renderListEl(attrs plugindata.Map) string {
	itemsRendered := attrs["items_rendered"].(plugindata.List)
	format := string(attrs["format"].(plugindata.String))

	var prefix string

	switch format {
	case "ordered":
		prefix = "1."
	case "tasklist":
		prefix = "- [ ]"
	default:
		prefix = "-"
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
	headers := attrs["headers"].(plugindata.List)
	rows := attrs["rows"].(plugindata.List)

	cellRows := utils.FnMap(rows, func(cellsData plugindata.Data) []string {
		cells, _ := cellsData.(plugindata.List)
		rowValues := []string{}
		for _, cellData := range cells {
			cell := cellData.(plugindata.Map)
			cellValue := string(cell["value"].(plugindata.String))
			rowValues = append(rowValues, cellValue)
		}

		return rowValues
	})

	// https://github.com/olekukonko/tablewriter?tab=readme-ov-file#2-markdown-table

	buf := &strings.Builder{}
	table := tablewriter.NewTable(
		buf,
		tablewriter.WithRenderer(renderer.NewMarkdown()),
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Formatting: tw.CellFormatting{AutoFormat: tw.Off},
				Alignment:  tw.CellAlignment{Global: tw.AlignCenter},
			},
			Row: tw.CellConfig{Alignment: tw.CellAlignment{Global: tw.AlignLeft}},
		}),
	)

	headerValues := headers.Any().([]any)
	table.Header(headerValues...)
	_ = table.Bulk(cellRows) // bytes.Buffer writes do not fail.
	_ = table.Render()       // bytes.Buffer writes do not fail.

	return buf.String()
}

func renderElement(el plugin.Content, depth int) string {
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

		if section.Title != nil {
			outputs = append(outputs, renderElement(section.Title, depth+1))
		}

		for _, child := range section.Children {
			outputs = append(outputs, renderElement(child, depth+1))
		}
		return strings.Join(outputs, joinBlocksWith)
	case plugin.EmptyKind:
		// nothing to add
	case plugin.TitleKind:
		output := renderHeadingEl(attrs)
		// MD022 - https://github.com/DavidAnson/markdownlint/blob/main/doc/md022.md
		// return fmt.Sprintf("\n%s\n", output)
		return output
	case plugin.TextKind:
		return renderTextEl(attrs)
	case plugin.BlockquoteKind:
		return renderQuoteEl(attrs)
	case plugin.ImageKind:
		return renderImageEl(attrs)
	case plugin.CodeKind:
		output := renderCodeEl(attrs)
		// MD031 - https://github.com/DavidAnson/markdownlint/blob/main/doc/md031.md
		// return fmt.Sprintf("\n%s\n", output)
		return output
	case plugin.ListKind:
		output := renderListEl(attrs)
		// MD032 - https://github.com/DavidAnson/markdownlint/blob/main/doc/md032.md
		// return fmt.Sprintf("\n%s\n", output)
		return output
	case plugin.TableKind:
		output := renderTableEl(attrs)
		// MD058 - https://github.com/DavidAnson/markdownlint/blob/main/doc/md058.md
		// return fmt.Sprintf("\n%s\n", output)
		return output
	case plugin.TOCKind:
		return renderTOCEl(attrs)
	}
	return fmt.Sprintf("<error-unknown-element: %s>", el.Kind())
}

func makeMarkdownFormatterFunc(log *slog.Logger, tracer trace.Tracer) plugin.FormatFunc {
	return func(ctx context.Context, params *plugin.FormatParams) (_ *plugin.FormattedContent, diags diagnostics.Diag) {
		dataCtx := params.DataContext
		dataCtx["format"] = plugindata.String(params.Format)

		section, err := ParseContentSection(params.Content)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse document content",
				Detail:   fmt.Sprintf("Error while parsing document content: %s", err),
			}}
		}

		outputs := []string{}

		if section.Title != nil {
			outputs = append(outputs, renderElement(section.Title, 0))
		}

		for _, child := range section.Children {
			out := renderElement(child, 0)
			outputs = append(outputs, out)
		}

		var frontmatter *string

		frontmatterVal := params.Args.GetAttrVal("frontmatter")
		if !frontmatterVal.IsNull() {

			jsonBytes, err := ctyjson.Marshal(frontmatterVal, frontmatterVal.Type())
			if err != nil {
				panic(err)
			}

			// 2. Unmarshal JSON into a standard Go map
			var rawMap map[string]any
			if err := json.Unmarshal(jsonBytes, &rawMap); err != nil {
				panic(err)
			}

			format := params.Args.GetAttrVal("frontmatter_format").AsString()

			var diag diagnostics.Diag
			frontmatter, diag = renderFrontmatter(format, rawMap)
			if diags.Extend(diag) {
				return nil, diags
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

func renderFrontmatter(format string, data map[string]any) (*string, diagnostics.Diag) {
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

func renderYAMLFrontMatter(m map[string]any) (string, error) {
	var buf strings.Builder
	buf.WriteString("---\n")
	err := yaml.NewEncoder(&buf).Encode(m)
	if err != nil {
		return "", err
	}
	buf.WriteString("---")
	return buf.String(), nil
}

func renderTOMLFrontMatter(m map[string]any) (string, error) {
	var buf strings.Builder
	buf.WriteString("+++\n")
	err := toml.NewEncoder(&buf).Encode(m)
	if err != nil {
		return "", err
	}
	buf.WriteString("+++")
	return buf.String(), nil
}

func renderJSONFrontMatter(m map[string]any) (string, error) {
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	err := enc.Encode(m)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
