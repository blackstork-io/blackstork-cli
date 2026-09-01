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
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	u "github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugins/builtin/utils"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

var supportedFileFormats = u.FnMap(
	[]string{
		"json",
		"yaml",
		"csv",
		"text",
	},
	cty.StringVal,
)

func makeFileDataSource(log *slog.Logger) *plugin.DataSource {
	return &plugin.DataSource{
		DataFunc: func(
			ctx context.Context,
			params *plugin.RetrieveDataParams,
		) (plugindata.Data, diagnostics.Diag) {
			return fetchFromFile(ctx, log, params)
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:       "glob",
					Type:       cty.String,
					ExampleVal: cty.StringVal("path/to/file*.json"),
					Doc:        `A glob pattern to select files to read`,
				},
				{
					Name:       "path",
					Type:       cty.String,
					ExampleVal: cty.StringVal("path/to/file.yaml"),
					Doc:        `A file path to a file to read`,
				},
				{
					Name:        "format",
					Type:        cty.String,
					ExampleVal:  cty.StringVal("json"),
					Constraints: constraint.RequiredMeaningful,
					OneOf:       constraint.OneOf(supportedFileFormats),
					Doc:         `File format`,
				},
				{
					Name:         "csv_delimiter",
					Type:         cty.String,
					DefaultVal:   cty.StringVal(","),
					MinInclusive: cty.NumberIntVal(1),
					MaxInclusive: cty.NumberIntVal(1),
					Doc:          `CSV field delimiter`,
				},
			},
		},
		Doc: u.Dedent(
			`
			Loads files with the names that match provided ` + "`glob`" + ` pattern or a single file from provided ` + "`path`" + `value.

			Either ` + "`glob`" + ` or ` + "`path`" + ` must be set.

			When ` + "`path`" + ` argument is specified, the data source returns only the content of a file.
			When ` + "`glob`" + ` argument is specified, the data source returns a list of dicts that contain the content of a file and file's metadata. For example:

			` + "```json" + `
			[
			  {
			    "file_path": "path/file-a.json",
			    "file_name": "file-a.json",
			    "content": {
			      "foo": "bar"
			  }
			  },
			  {
			    "file_path": "path/file-b.json",
			    "file_name": "file-b.json",
			    "content": [
			      {"x": "y"}
			    ]
			  }
			]
			` + "```" + `

			If multiple files are matched, all files must have the same file format.

			For CSV files, the data source assumes that CSV file has a header: each line of the file is turned into a map with the column titles as keys.

			For example, CSV file with the following data:

			| column_A | column_B | column_C |
			| -------- | -------- | -------- |
			| Foo      | true     | 42       |
			| Bar      | false    | 4.2      |

			will be represented as the following data structure:
			` + "```json" + `
			[
			  {"column_A": "Foo", "column_B": true, "column_C": 42},
			  {"column_A": "Bar", "column_B": false, "column_C": 4.2}
			]
			` + "```" + `
			`,
		),
	}
}

func fetchFromFile(
	ctx context.Context,
	log *slog.Logger,
	params *plugin.RetrieveDataParams,
) (plugindata.Data, diagnostics.Diag) {
	glob := params.Args.GetAttrVal("glob")
	path := params.Args.GetAttrVal("path")
	format := params.Args.GetAttrVal("format").AsString()

	var err error
	var csvDelimiter *rune

	if delimiterVal := params.Config.GetAttrVal("csv_delimiter"); !delimiterVal.IsNull() {
		csvDelimiter, err = getCSVDelim(delimiterVal.AsString())
		if err != nil {
			log.ErrorContext(ctx, "Error while getting a delimiter value", "err", err)
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("%s", err),
			}}
		}
	}

	if !path.IsNull() && path.AsString() != "" {
		pathStr := path.AsString()
		log.DebugContext(ctx, "Reading file from path", "path", pathStr)
		data, err := readAndDecodeFile(ctx, pathStr, format, csvDelimiter)
		if err != nil {
			log.ErrorContext(
				ctx, "Error while reading file",
				"path", pathStr,
				"err", err,
			)
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to read file",
				Detail:   err.Error(),
			}}
		}
		return data, nil
	} else if !glob.IsNull() && glob.AsString() != "" {
		globStr := glob.AsString()
		log.DebugContext(ctx, "Reading the files that match glob", "glob", globStr)
		data, err := readFiles(ctx, globStr, format, csvDelimiter)
		if err != nil {
			slog.Error(
				"Error while reading files",
				"glob", globStr,
				"err", err,
			)
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to read files",
				Detail:   err.Error(),
			}}
		}
		return data, nil
	}
	log.ErrorContext(ctx, `Neither "glob" nor "path" were provided`)
	return nil, diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Invalid arguments",
		Detail:   `Either "glob" value or "path" value must be provided`,
	}}
}

func readAndDecodeFile(
	ctx context.Context,
	path string,
	format string,
	csvDelim *rune,
) (plugindata.Data, error) {
	switch format {
	case "json":
		return readAndDecodeJSONFile(path)
	case "yaml":
		return readAndDecodeYAMLFile(path)
	case "csv":
		if csvDelim == nil {
			csvDelim = new(',')
		}
		return readAndDecodeCSVFile(ctx, path, *csvDelim)
	case "text":
		txtData, err := os.ReadFile(path) //nolint:gosec // Reading user-configured paths is the purpose of this data source.
		if err != nil {
			return nil, err
		}
		return plugindata.String(string(txtData)), nil
	default:
		return nil, fmt.Errorf("unsupported file format: %s", format)
	}
}

func readAndDecodeCSVFile(
	ctx context.Context,
	path string,
	delimiter rune,
) (plugindata.List, error) {
	f, err := os.Open(path) //nolint:gosec // Reading user-configured paths is the purpose of this data source.
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.Comma = delimiter

	return utils.ParseCSVContent(ctx, reader)
}

func readAndDecodeJSONFile(path string) (plugindata.Data, error) {
	jsonData, err := os.ReadFile(path) //nolint:gosec // Reading user-configured paths is the purpose of this data source.
	if err != nil {
		return nil, err
	}
	return plugindata.UnmarshalJSON(jsonData)
}

func readAndDecodeYAMLFile(path string) (plugindata.Data, error) {
	yamlData, err := os.ReadFile(path) //nolint:gosec // Reading user-configured paths is the purpose of this data source.
	if err != nil {
		return nil, err
	}
	return plugindata.UnmarshalYAML(yamlData)
}

func readFiles(
	ctx context.Context,
	pattern string,
	format string,
	csvDelimiter *rune,
) (plugindata.List, error) {
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	result := make(plugindata.List, 0, len(paths))
	for _, path := range paths {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			content, err := readAndDecodeFile(ctx, path, format, csvDelimiter)
			if err != nil {
				return result, err
			}
			result = append(result, plugindata.Map{
				"file_path": plugindata.String(path),
				"file_name": plugindata.String(filepath.Base(path)),
				"content":   content,
			})
		}
	}
	return result, nil
}

func getCSVDelim(delim string) (*rune, error) {
	delimRune, runeLen := utf8.DecodeRuneInString(delim)
	if runeLen == 0 || len(delim) != runeLen {
		return nil, errors.New("delimiter must be a single character")
	}
	return &delimRune, nil
}

// type jsonData struct {
// 	data plugindata.Data
// }
//
// func (d jsonData) toData(v any) (res plugindata.Data, err error) {
// 	return plugindata.ParseAny(v)
// }
//
// func (d *jsonData) UnmarshalJSON(b []byte) error {
// 	if !json.Valid(b) {
// 		return fmt.Errorf("invalid JSON data")
// 	}
// 	var result any
// 	err := json.Unmarshal(b, &result)
// 	if err != nil {
// 		return err
// 	}
// 	d.data, err = d.toData(result)
// 	return err
// }
