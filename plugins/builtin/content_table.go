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
	"strings"
	"text/template"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

// type tableCellTmpl = *template.Template

func makeTableContentProvider() *plugin.ContentProvider {
	return &plugin.ContentProvider{
		ContentFunc: genTableContent,
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "rows",
					Type: cty.List(plugindata.Encapsulated.CtyType()),
					Doc:  "A list of objects representing rows in the table. Can be a static list or `query_jq()` func call.",
				},
				{
					Name: "columns",
					Type: cty.List(cty.Object(map[string]cty.Type{
						"header": cty.String,
						"value":  cty.String,
					})),
					Doc: `List of a header and a cell template pairs for each column`,
					ExampleVal: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"header": cty.StringVal("1st column header template"),
							"value":  cty.StringVal("1st column value template"),
						}),
						cty.ObjectVal(map[string]cty.Value{
							"header": cty.StringVal("2nd column header template"),
							"value":  cty.StringVal("2nd column value template"),
						}),
					}),
					Constraints: constraint.RequiredMeaningful,
				},
			},
		},
		Doc: `
			Renders data as a table using a Go template for each column header and cell.

			Each cell template has access to the document data context and these variables:

			* ` + "`.rows` – the value of `rows` argument" + `
			* ` + "`.row.value` – the current row from `.rows` list" + `
			* ` + "`.row.index` – the current row index" + `
			* ` + "`.col.index` – the current column index" + `

			Header templates have access to the document data context, ` + "`.rows`" + `, and ` + "`.col.index`" + `,
			but not ` + "`.row.value` or `.row.index`" + `.`,
	}
}

func genTableContent(
	ctx context.Context,
	params *plugin.ProvideContentParams,
) (*plugin.ContentProviderResult, error) {
	var rows plugindata.List
	rowsVal := params.Args.GetAttrVal("rows")
	if !rowsVal.IsNull() {
		var err error
		rows, err = utils.FnMapErr(rowsVal.AsValueSlice(), func(v cty.Value) (plugindata.Data, error) {
			data, err := plugindata.Encapsulated.FromCty(v)
			if err != nil {
				return nil, err
			}
			if data == nil {
				return nil, nil
			}
			return *data, nil
		})
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse `rows` attribute",
				Detail:   err.Error(),
				Subject:  &params.Args.Attrs["rows"].ValueRange,
			}}
		}
	}

	columnTemplates := params.Args.GetAttrVal("columns").AsValueSlice()

	var headerTemplates []*template.Template
	var cellTemplates []*template.Template

	for _, columnTemplatePair := range columnTemplates {
		pair := columnTemplatePair.AsValueMap()

		header := pair["header"]
		if header.IsNull() {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "No `header` template provided for the column",
				Detail:   "The column misses `header` template",
				Subject:  &params.Args.Attrs["columns"].ValueRange,
			}}
		}

		value := pair["value"]
		if value.IsNull() {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "No `value` template provided for the column",
				Detail:   "The column misses `value` template",
				Subject:  &params.Args.Attrs["columns"].ValueRange,
			}}
		}

		headerTmpl, err := prepareTemplate("header", header.AsString())
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse a header template",
				Detail:   err.Error(),
				Subject:  &params.Args.Attrs["columns"].ValueRange,
			}}
		}

		valueTmpl, err := prepareTemplate("value", value.AsString())
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse a cell value template",
				Detail:   err.Error(),
				Subject:  &params.Args.Attrs["columns"].ValueRange,
			}}
		}

		headerTemplates = append(headerTemplates, headerTmpl)
		cellTemplates = append(cellTemplates, valueTmpl)
	}

	data := params.DataContext.Any().(map[string]any)
	data["rows"] = rows.Any()

	var headerBuf bytes.Buffer

	var headers []string
	for i, headerTempl := range headerTemplates {
		headerBuf.Reset()
		data["col"] = map[string]any{"index": i + 1}
		err := headerTempl.Execute(&headerBuf, data)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to render header template",
				Detail:   err.Error(),
				Subject:  &params.Args.Attrs["columns"].ValueRange,
			}}
		}
		delete(data, "col")
		header := strings.TrimSpace(headerBuf.String())
		headers = append(headers, header)
	}

	results := make(plugindata.List, len(rows))

	var cellBuf bytes.Buffer
	for rowIdx, row := range rows {

		rowResult := make(plugindata.List, len(cellTemplates))

		rowData := map[string]any{}
		rowData["index"] = rowIdx + 1
		rowData["value"] = row.Any()

		data["row"] = rowData

		for colIdx, cellTemplate := range cellTemplates {
			cellBuf.Reset()

			colData := map[string]any{}
			data["col"] = colData

			colData["index"] = colIdx + 1
			colData["header"] = headers[colIdx]

			err := cellTemplate.Execute(&cellBuf, data)
			if err != nil {
				return nil, diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Failed to render cell template",
					Detail:   err.Error(),
					Subject:  &params.Args.Attrs["columns"].ValueRange,
				}}
			}

			cellVal := strings.TrimSpace(cellBuf.String())
			// cellDataMap, err := plugindata.ParseMapAny(data)
			// if err != nil {
			// 	return nil, diagnostics.Diag{{
			// 		Severity: hcl.DiagError,
			// 		Summary:  "Failed to prepare data context for cell",
			// 		Detail:   err.Error(),
			// 	}}
			// }
			colDataData, err := plugindata.ParseMapAny(colData)
			if err != nil {
				return nil, diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Failed to convert col data",
					Detail:   err.Error(),
				}}
			}

			cellResult := plugindata.Map{
				"value": plugindata.String(cellVal),
				"col":   colDataData,
				// Adding full data to every cell is excessive
				// "data":  cellDataMap,
			}
			rowResult[colIdx] = cellResult
		}

		results[rowIdx] = rowResult
	}

	headerValues := utils.FnMap(headers, func(h string) plugindata.Data {
		return plugindata.String(h)
	})
	headersData := plugindata.List(headerValues)

	return &plugin.ContentProviderResult{
		Content: plugin.NewTableElement(headersData, results),
	}, nil
}

// func renderTableContent(
// 	headers, values []tableCellTmpl,
// 	dataCtx plugindata.Map,
// 	rowsList plugindata.List,
// ) (string, error) {
// 	var buf bytes.Buffer
//
// 	data := dataCtx.Any().(map[string]any)
//
// 	rows := rowsList.Any().([]any)
// 	data["rows"] = rows
// 	col := map[string]any{}
// 	data["col"] = col
// 	buf.WriteByte('|')
// 	var cellBuf bytes.Buffer
// 	for colIdx, header := range headers {
// 		cellBuf.Reset()
// 		col["index"] = colIdx + 1
// 		err := header.Execute(&cellBuf, data)
// 		if err != nil {
// 			return "", fmt.Errorf("failed to render header: %w", err)
// 		}
//
// 		buf.Write(
// 			bytes.ReplaceAll(
// 				bytes.TrimSpace(cellBuf.Bytes()),
// 				[]byte("\n"),
// 				[]byte(" "),
// 			),
// 		)
// 		buf.WriteByte('|')
// 	}
// 	buf.WriteByte('\n')
// 	buf.WriteByte('|')
// 	for range headers {
// 		buf.WriteString("---|")
// 	}
// 	buf.WriteString("\n")
//
// 	dataRow := map[string]any{}
// 	data["row"] = dataRow
//
// 	for rowIdx, row := range rows {
// 		buf.WriteByte('|')
// 		dataRow["index"] = rowIdx + 1
// 		dataRow["value"] = row
// 		for colIdx, value := range values {
// 			cellBuf.Reset()
// 			col["index"] = colIdx + 1
// 			err := value.Execute(&cellBuf, data)
// 			if err != nil {
// 				return "", fmt.Errorf("failed to render value: %w", err)
// 			}
// 			buf.Write(
// 				bytes.ReplaceAll(
// 					bytes.TrimSpace(cellBuf.Bytes()),
// 					[]byte("\n"),
// 					[]byte(" "),
// 				),
// 			)
// 			buf.WriteByte('|')
// 		}
// 		buf.WriteByte('\n')
// 	}
//
// 	return buf.String(), nil
// }
