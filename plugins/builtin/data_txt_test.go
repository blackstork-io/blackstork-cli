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

func Test_makeTXTDataSchema(t *testing.T) {
	schema := makeTXTDataSource()
	assert.Nil(t, schema.Config)
	assert.NotNil(t, schema.Args)
	assert.NotNil(t, schema.DataFunc)
}

func Test_fetchTXTData(t *testing.T) {
	tt := []struct {
		name          string
		path          string
		glob          string
		expectedData  plugindata.Data
		expectedDiags diagtest.Asserts
	}{
		{
			name:         "valid_path",
			path:         filepath.Join("testdata", "txt", "data.txt"),
			expectedData: plugindata.String("data_content"),
		},
		{
			name: "with_glob_matches",
			glob: filepath.Join("testdata", "txt", "dat*.txt"),
			expectedData: plugindata.List{
				plugindata.Map{
					"file_name": plugindata.String("data.txt"),
					"file_path": plugindata.String(filepath.Join("testdata", "txt", "data.txt")),
					"content":   plugindata.String("data_content"),
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
			name:         "no_glob_matches",
			glob:         filepath.Join("testdata", "txt", "does-not-exist*.txt"),
			expectedData: plugindata.List{},
		},
		{
			name: "invalid_path",
			path: filepath.Join("testdata", "txt", "does_not_exist.txt"),
			expectedDiags: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryEquals("Failed to read a file"),
				diagtest.DetailContains("Failed to open a file", "does_not_exist.txt"),
			}},
		},
		{
			name:         "empty_file_with_path",
			path:         filepath.Join("testdata", "txt", "empty.txt"),
			expectedData: plugindata.String(""),
		},
		{
			name: "empty_file_with_glob",
			glob: filepath.Join("testdata", "txt", "empt*.txt"),
			expectedData: plugindata.List{
				plugindata.Map{
					"file_name": plugindata.String("empty.txt"),
					"file_path": plugindata.String(filepath.Join("testdata", "txt", "empty.txt")),
					"content":   plugindata.String(""),
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			p := &plugin.Schema{
				DataSources: plugin.DataSources{
					"txt": makeTXTDataSource(),
				},
			}

			args := plugintest.NewTestDecoder(t, p.DataSources["txt"].Args)
			if tc.path != "" {
				args.SetAttr("path", cty.StringVal(tc.path))
			}
			if tc.glob != "" {
				args.SetAttr("glob", cty.StringVal(tc.glob))
			}

			argVal, fm, diags := args.DecodeDiagFiles()

			var (
				diag diagnostics.Diag
				data plugindata.Data
			)
			if !diags.HasErrors() {
				ctx := context.Background()
				data, diag = p.RetrieveData(ctx, "txt", &plugin.RetrieveDataParams{Args: argVal})
				diags.Extend(diag)
			}
			assert.Equal(t, tc.expectedData, data)
			tc.expectedDiags.AssertMatch(t, diags, fm)
		})
	}
}
