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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugin/plugintest"
)

func Test_makeYAMLDataSchema(t *testing.T) {
	schema := makeYAMLDataSource()
	assert.Nil(t, schema.Config)
	assert.NotNil(t, schema.Args)
	assert.NotNil(t, schema.DataFunc)
}

func Test_fetchYAMLData(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	tt := []struct {
		name          string
		glob          string
		path          string
		expectedData  plugindata.Data
		expectedDiags diagtest.Asserts
	}{
		{
			name: "invalid_yaml_file",
			glob: filepath.Join("testdata", "yaml", "invalid.*"),
			expectedDiags: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryEquals("Failed to read the files"),
				diagtest.DetailContains("yaml: line 2: could not find expected ':'"),
			}},
		},
		{
			name: "invalid_yaml_file_with_path",
			path: filepath.Join("testdata", "yaml", "invalid.txt"),
			expectedDiags: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryEquals("Failed to read the file"),
				diagtest.DetailContains("yaml: line 2: could not find expected ':'"),
			}},
		},
		{
			name: "no_params",
			expectedDiags: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryEquals("Failed to parse provided arguments"),
				diagtest.DetailEquals("Either \"glob\" value or \"path\" value must be provided"),
			}},
		},
		{
			name:         "no_glob_matches",
			glob:         filepath.Join("testdata", "yaml", "unknown_dir", "*.yaml"),
			expectedData: plugindata.List{},
		},
		{
			name: "no_path_match",
			path: filepath.Join("testdata", "yaml", "unknown_dir", "does-not-exist.yaml"),
			expectedDiags: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryEquals("Failed to read the file"),
				diagtest.DetailContains("open", "does-not-exist.yaml"),
			}},
		},
		{
			name: "load_one_file_with_path",
			path: filepath.Join("testdata", "yaml", "a.yaml"),
			expectedData: plugindata.Map{
				"property_for": plugindata.String("a.yaml"),
			},
		},
		{
			name: "glob_matches_one_file",
			glob: filepath.Join("testdata", "yaml", "a.yaml"),
			expectedData: plugindata.List{
				plugindata.Map{
					"file_name": plugindata.String("a.yaml"),
					"file_path": plugindata.String(filepath.Join("testdata", "yaml", "a.yaml")),
					"content": plugindata.Map{
						"property_for": plugindata.String("a.yaml"),
					},
				},
			},
		},
		{
			name: "glob_matches_multiple_files",
			glob: filepath.Join("testdata", "yaml", "dir", "*.yaml"),
			expectedData: plugindata.List{
				plugindata.Map{
					"file_name": plugindata.String("b.yaml"),
					"file_path": plugindata.String(filepath.Join("testdata", "yaml", "dir", "b.yaml")),
					"content": plugindata.List{
						plugindata.Map{
							"id":           plugindata.Number(1),
							"property_for": plugindata.String("dir/b.yaml"),
						},
						plugindata.Map{
							"id":           plugindata.Number(2),
							"property_for": plugindata.String("dir/b.yaml"),
						},
					},
				},
				plugindata.Map{
					"file_name": plugindata.String("c.yaml"),
					"file_path": plugindata.String(filepath.Join("testdata", "yaml", "dir", "c.yaml")),
					"content": plugindata.List{
						plugindata.Map{
							"id":           plugindata.Number(3),
							"property_for": plugindata.String("dir/c.yaml"),
						},
						plugindata.Map{
							"id":           plugindata.Number(4),
							"property_for": plugindata.String("dir/c.yaml"),
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			p := &plugin.Schema{
				DataSources: plugin.DataSources{
					"yaml": makeYAMLDataSource(),
				},
			}

			args := plugintest.NewTestDecoder(t, p.DataSources["yaml"].Args)
			if tc.path != "" {
				args.SetAttr("path", cty.StringVal(tc.path))
			}
			if tc.glob != "" {
				args.SetAttr("glob", cty.StringVal(tc.glob))
			}

			argVal, fm, diags := args.DecodeDiagFiles()

			var (
				data plugindata.Data
				diag diagnostics.Diag
			)
			if !diags.HasErrors() {
				ctx := context.Background()
				data, diag = p.RetrieveData(ctx, "yaml", &plugin.RetrieveDataParams{Args: argVal})

				slog.Info("WHAT1", "data", data)
				slog.Info("WHAT2", "diag", diag)

				diags.Extend(diag)
			}
			assert.Equal(t, tc.expectedData, data)
			tc.expectedDiags.AssertMatch(t, diags, fm)
		})
	}
}

func Test_readYAMLFilesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	data, err := readYAMLFiles(ctx, filepath.Join("testdata", "yaml", "a.yaml"))
	assert.Nil(t, data)
	assert.Error(t, context.Canceled, err)
}
