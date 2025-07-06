package builtin

import (
	"context"
	"log/slog"
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

func Test_makeJSONDataSchema(t *testing.T) {
	schema := makeJSONDataSource()
	assert.Nil(t, schema.Config)
	assert.NotNil(t, schema.Args)
	assert.NotNil(t, schema.DataFunc)
}

func Test_fetchJSONData(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	tt := []struct {
		name          string
		glob          string
		path          string
		expectedData  plugindata.Data
		expectedDiags diagtest.Asserts
	}{
		{
			name: "invalid_json_file",
			glob: filepath.Join("testdata", "json", "invalid.txt"),
			expectedDiags: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryEquals("Failed to read the files"),
				diagtest.DetailEquals("invalid character 'i' looking for beginning of object key string"),
			}},
		},
		{
			name: "invalid_json_file_with_path",
			path: filepath.Join("testdata", "json", "invalid.txt"),
			expectedDiags: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryEquals("Failed to read the file"),
				diagtest.DetailEquals("invalid character 'i' looking for beginning of object key string"),
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
			glob:         filepath.Join("testdata", "json", "unknown_dir", "*.json"),
			expectedData: plugindata.List{},
		},
		{
			name: "no_path_match",
			path: filepath.Join("testdata", "json", "unknown_dir", "does-not-exist.json"),
			expectedDiags: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryEquals("Failed to read the file"),
				diagtest.DetailContains("open", "does-not-exist.json"),
			}},
		},
		{
			name: "load_one_file_with_path",
			path: filepath.Join("testdata", "json", "a.json"),
			expectedData: plugindata.Map{
				"property_for": plugindata.String("a.json"),
			},
		},
		{
			name: "glob_matches_one_file",
			glob: filepath.Join("testdata", "json", "a.json"),
			expectedData: plugindata.List{
				plugindata.Map{
					"file_name": plugindata.String("a.json"),
					"file_path": plugindata.String(filepath.Join("testdata", "json", "a.json")),
					"content": plugindata.Map{
						"property_for": plugindata.String("a.json"),
					},
				},
			},
		},
		{
			name: "glob_matches_multiple_files",
			glob: filepath.Join("testdata", "json", "dir", "*.json"),
			expectedData: plugindata.List{
				plugindata.Map{
					"file_name": plugindata.String("b.json"),
					"file_path": plugindata.String(filepath.Join("testdata", "json", "dir", "b.json")),
					"content": plugindata.List{
						plugindata.Map{
							"id":           plugindata.Number(1),
							"property_for": plugindata.String("dir/b.json"),
						},
						plugindata.Map{
							"id":           plugindata.Number(2),
							"property_for": plugindata.String("dir/b.json"),
						},
					},
				},
				plugindata.Map{
					"file_name": plugindata.String("c.json"),
					"file_path": plugindata.String(filepath.Join("testdata", "json", "dir", "c.json")),
					"content": plugindata.List{
						plugindata.Map{
							"id":           plugindata.Number(3),
							"property_for": plugindata.String("dir/c.json"),
						},
						plugindata.Map{
							"id":           plugindata.Number(4),
							"property_for": plugindata.String("dir/c.json"),
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
					"json": makeJSONDataSource(),
				},
			}

			args := plugintest.NewTestDecoder(t, p.DataSources["json"].Args)
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
				data, diag = p.RetrieveData(ctx, "json", &plugin.RetrieveDataParams{Args: argVal})
				diags.Extend(diag)
			}
			assert.Equal(t, tc.expectedData, data)
			tc.expectedDiags.AssertMatch(t, diags, fm)
		})
	}
}

func Test_readJSONFilesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	data, err := readJSONFiles(ctx, filepath.Join("testdata", "json", "a.json"))
	assert.Nil(t, data)
	assert.Error(t, context.Canceled, err)
}
