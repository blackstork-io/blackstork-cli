package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/utils"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

func makeJSONDataSource() *plugin.DataSource {
	return &plugin.DataSource{
		DataFunc: fetchJSONData,
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:       "glob",
					Type:       cty.String,
					ExampleVal: cty.StringVal("path/to/file*.json"),
					Doc:        `A glob pattern to select JSON files to read`,
				},
				{
					Name:       "path",
					Type:       cty.String,
					ExampleVal: cty.StringVal("path/to/file.json"),
					Doc:        `A file path to a JSON file to read`,
				},
			},
		},
		Doc: utils.Dedent(`
			Loads JSON files with the names that match provided ` + "`glob`" + ` pattern or a single file from provided ` + "`path`" + `value.

			Either ` + "`glob`" + ` or ` + "`path`" + ` argument must be set.

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
			` + "```",
		),
	}
}

func fetchJSONData(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
	glob := params.Args.GetAttrVal("glob")
	path := params.Args.GetAttrVal("path")

	if !path.IsNull() && path.AsString() != "" {
		slog.Debug("Reading a file from a path", "path", path.AsString())
		data, err := readAndDecodeJSONFile(path.AsString())
		if err != nil {
			slog.Error(
				"Error while reading a JSON file",
				slog.String("path", path.AsString()),
				slog.Any("error", err),
			)
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to read the file",
				Detail:   err.Error(),
			}}
		}
		return data, nil
	} else if !glob.IsNull() && glob.AsString() != "" {
		slog.Debug("Reading the files that match the glob pattern", "glob", glob.AsString())
		data, err := readJSONFiles(ctx, glob.AsString())
		if err != nil {
			slog.Error(
				"Error while reading the JSON files",
				slog.String("glob", glob.AsString()),
				slog.Any("error", err),
			)
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to read the files",
				Detail:   err.Error(),
			}}
		}
		return data, nil
	}
	slog.Error("Either \"glob\" value or \"path\" value must be provided")
	return nil, diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to parse provided arguments",
		Detail:   "Either \"glob\" value or \"path\" value must be provided",
	}}
}

func readAndDecodeJSONFile(path string) (plugindata.Data, error) {
	jsonData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return plugindata.UnmarshalJSON(jsonData)
}

func readJSONFiles(ctx context.Context, pattern string) (plugindata.List, error) {
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
			content, err := readAndDecodeJSONFile(path)
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

type jsonData struct {
	data plugindata.Data
}

func (d jsonData) toData(v any) (res plugindata.Data, err error) {
	return plugindata.ParseAny(v)
}

func (d *jsonData) UnmarshalJSON(b []byte) error {
	if !json.Valid(b) {
		return fmt.Errorf("invalid JSON data")
	}
	var result any
	err := json.Unmarshal(b, &result)
	if err != nil {
		return err
	}
	d.data, err = d.toData(result)
	return err
}
