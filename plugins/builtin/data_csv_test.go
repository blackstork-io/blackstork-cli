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
	"maps"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugin/plugintest"
)

func Test_makeCSVDataSchema(t *testing.T) {
	schema := makeCSVDataSource()
	require.NotNil(t, schema, "expected data source csv")
	assert.NotNil(t, schema.DataFunc)
	assert.NotNil(t, schema.Args)
	assert.NotNil(t, schema.Config)
}

func Test_fetchCSVData(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	tt := []struct {
		name          string
		path          string
		glob          string
		delimiter     string
		expectedData  plugindata.Data
		expectedDiags diagtest.Asserts
	}{
		{
			name:      "comma_delim_path",
			path:      filepath.Join("testdata", "csv", "comma.csv"),
			delimiter: ",",
			expectedData: plugindata.List{
				plugindata.Map{
					"id":     plugindata.String("b8fa4bb0-6dd4-45ba-96e0-9a182b2b932e"),
					"active": plugindata.Bool(true),
					"name":   plugindata.String("Stacey"),
					"age":    plugindata.Number(26),
					"height": plugindata.Number(1.98),
				},
				plugindata.Map{
					"id":     plugindata.String("b0086c49-bcd8-4aae-9f88-4f46b128e709"),
					"active": plugindata.Bool(false),
					"name":   plugindata.String("Myriam"),
					"age":    plugindata.Number(33),
					"height": plugindata.Number(1.81),
				},
				plugindata.Map{
					"id":     plugindata.String("a12d2a8c-eebc-42b3-be52-1ab0a2969a81"),
					"active": plugindata.Bool(true),
					"name":   plugindata.String("Oralee"),
					"age":    plugindata.Number(31),
					"height": plugindata.Number(2.23),
				},
			},
		},
		{
			name:      "semicolon_delim_path",
			path:      filepath.Join("testdata", "csv", "semicolon.csv"),
			delimiter: ";",
			expectedData: plugindata.List{
				plugindata.Map{
					"id":     plugindata.String("b8fa4bb0-6dd4-45ba-96e0-9a182b2b932e"),
					"active": plugindata.Bool(true),
					"name":   plugindata.String("Stacey"),
					"age":    plugindata.Number(26),
					"height": plugindata.Number(1.98),
				},
				plugindata.Map{
					"id":     plugindata.String("b0086c49-bcd8-4aae-9f88-4f46b128e709"),
					"active": plugindata.Bool(false),
					"name":   plugindata.String("Myriam"),
					"age":    plugindata.Number(33),
					"height": plugindata.Number(1.81),
				},
				plugindata.Map{
					"id":     plugindata.String("a12d2a8c-eebc-42b3-be52-1ab0a2969a81"),
					"active": plugindata.Bool(true),
					"name":   plugindata.String("Oralee"),
					"age":    plugindata.Number(31),
					"height": plugindata.Number(2.23),
				},
			},
		},
		{
			name:      "comma_delim_glob",
			glob:      filepath.Join("testdata", "csv", "comm*.csv"),
			delimiter: ",",
			expectedData: plugindata.List{
				plugindata.Map{
					"file_name": plugindata.String("comma.csv"),
					"file_path": plugindata.String(filepath.Join("testdata", "csv", "comma.csv")),
					"content": plugindata.List{
						plugindata.Map{
							"id":     plugindata.String("b8fa4bb0-6dd4-45ba-96e0-9a182b2b932e"),
							"active": plugindata.Bool(true),
							"name":   plugindata.String("Stacey"),
							"age":    plugindata.Number(26),
							"height": plugindata.Number(1.98),
						},
						plugindata.Map{
							"id":     plugindata.String("b0086c49-bcd8-4aae-9f88-4f46b128e709"),
							"active": plugindata.Bool(false),
							"name":   plugindata.String("Myriam"),
							"age":    plugindata.Number(33),
							"height": plugindata.Number(1.81),
						},
						plugindata.Map{
							"id":     plugindata.String("a12d2a8c-eebc-42b3-be52-1ab0a2969a81"),
							"active": plugindata.Bool(true),
							"name":   plugindata.String("Oralee"),
							"age":    plugindata.Number(31),
							"height": plugindata.Number(2.23),
						},
					},
				},
			},
		},
		{
			name: "no_path_no_glob",
			expectedDiags: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryEquals("Failed to parse provided arguments"),
				diagtest.DetailEquals("Either \"glob\" value or \"path\" value must be provided"),
			}},
		},
		{
			name:      "invalid_delimiter",
			path:      filepath.Join("testdata", "csv", "comma.csv"),
			delimiter: "abc",
			expectedDiags: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.DetailContains(
					`The length`, `"delimiter"`, `exactly 1`,
				),
			}},
		},
		{
			name: "default_delimiter",
			path: filepath.Join("testdata", "csv", "comma.csv"),
			expectedData: plugindata.List{
				plugindata.Map{
					"id":     plugindata.String("b8fa4bb0-6dd4-45ba-96e0-9a182b2b932e"),
					"active": plugindata.Bool(true),
					"name":   plugindata.String("Stacey"),
					"age":    plugindata.Number(26),
					"height": plugindata.Number(1.98),
				},
				plugindata.Map{
					"id":     plugindata.String("b0086c49-bcd8-4aae-9f88-4f46b128e709"),
					"active": plugindata.Bool(false),
					"name":   plugindata.String("Myriam"),
					"age":    plugindata.Number(33),
					"height": plugindata.Number(1.81),
				},
				plugindata.Map{
					"id":     plugindata.String("a12d2a8c-eebc-42b3-be52-1ab0a2969a81"),
					"active": plugindata.Bool(true),
					"name":   plugindata.String("Oralee"),
					"age":    plugindata.Number(31),
					"height": plugindata.Number(2.23),
				},
			},
		},
		{
			name: "invalid_path",
			path: filepath.Join("testdata", "csv", "does_not_exist.csv"),
			expectedDiags: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("Failed to read a file"),
				diagtest.DetailContains("does_not_exist.csv"),
			}},
		},

		{
			name: "invalid_csv",
			path: filepath.Join("testdata", "csv", "invalid.csv"),
			expectedDiags: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("Failed to read a file"),
				diagtest.DetailEquals("record on line 2: wrong number of fields"),
			}},
		},
		{
			name:         "empty_csv",
			path:         filepath.Join("testdata", "csv", "empty.csv"),
			delimiter:    ",",
			expectedData: plugindata.List{},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			p := &plugin.Schema{
				DataSources: plugin.DataSources{
					"csv": makeCSVDataSource(),
				},
			}
			cfg := plugintest.NewTestDecoder(t, p.DataSources["csv"].Config)

			if tc.delimiter != "" {
				cfg.SetAttr("delimiter", cty.StringVal(tc.delimiter))
			}

			args := plugintest.NewTestDecoder(t, p.DataSources["csv"].Args)
			if tc.path != "" {
				args.SetAttr("path", cty.StringVal(tc.path))
			}
			if tc.glob != "" {
				args.SetAttr("glob", cty.StringVal(tc.glob))
			}
			argVal, fm, diags := args.DecodeDiagFiles()

			cfgVal, fm2, diag := cfg.DecodeDiagFiles()
			maps.Copy(fm, fm2)
			diags.Extend(diag)

			var data plugindata.Data
			if !diags.HasErrors() {
				ctx := context.Background()
				data, diag = p.RetrieveData(ctx, "csv", &plugin.RetrieveDataParams{Config: cfgVal, Args: argVal})
				diags.Extend(diag)
			}
			assert.Equal(t, tc.expectedData, data)
			tc.expectedDiags.AssertMatch(t, diags, fm)
		})
	}
}

func Test_readCSVFileCancellation(t *testing.T) {
	const defaultCSVDelimiter = ','
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	data, err := readAndDecodeCSVFile(ctx, filepath.Join("testdata", "csv", "comma.csv"), defaultCSVDelimiter)
	assert.Nil(t, data)
	assert.Error(t, context.Canceled, err)
}
