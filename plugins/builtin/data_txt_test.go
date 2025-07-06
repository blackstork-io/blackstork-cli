package builtin

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics/diagtest"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
	"github.com/blackstork-io/fabric/internal/plugin/plugintest"
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
