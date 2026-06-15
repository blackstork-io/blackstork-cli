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
	"log/slog"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

func makeYAMLDataSource() *plugin.DataSource {
	return &plugin.DataSource{
		DataFunc: fetchYAMLData,
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:       "glob",
					Type:       cty.String,
					ExampleVal: cty.StringVal("path/to/file*.yaml"),
					Doc:        `A glob pattern to select YAML files to read`,
				},
				{
					Name:       "path",
					Type:       cty.String,
					ExampleVal: cty.StringVal("path/to/file.yaml"),
					Doc:        `A file path to a YAML file to read`,
				},
			},
		},
		Doc: `
		Loads YAML files with the names that match provided ` + "`glob`" + ` pattern or a single file from provided ` + "`path`" + `value.

		Either ` + "`glob`" + ` or ` + "`path`" + ` argument must be set.

		When ` + "`path`" + ` argument is specified, the data source returns only the content of a file.
		When ` + "`glob`" + ` argument is specified, the data source returns a list of dicts that contain the content of a file and file's metadata. For example:

		` + "```json" + `
		[
		  {
			"file_path": "path/file-a.yaml",
			"file_name": "file-a.yaml",
			"content": {
			  "foo": "bar"
			}
		  },
		  {
			"file_path": "path/file-b.yaml",
			"file_name": "file-b.yaml",
			"content": [
			  {"x": "y"}
			]
		  }
		]
		` + "```",
	}
}

func fetchYAMLData(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
	glob := params.Args.GetAttrVal("glob")
	path := params.Args.GetAttrVal("path")

	if !path.IsNull() && path.AsString() != "" && !glob.IsNull() && glob.AsString() != "" {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse provided arguments",
			Detail:   "Either \"glob\" or \"path\" must be provided, not both",
		}}
	} else if !path.IsNull() && path.AsString() != "" {
		slog.Debug("Reading a file from a path", "path", path.AsString())
		data, err := readAndDecodeYAMLFile(path.AsString())
		if err != nil {
			slog.Error(
				"Error while reading a YAML file",
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
		data, err := readYAMLFiles(ctx, glob.AsString())
		if err != nil {
			slog.Error(
				"Error while reading the YAML files",
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

func readAndDecodeYAMLFile(path string) (plugindata.Data, error) {
	yamlData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return plugindata.UnmarshalYAML(yamlData)
}

func readYAMLFiles(ctx context.Context, pattern string) (plugindata.List, error) {
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
			content, err := readAndDecodeYAMLFile(path)
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
