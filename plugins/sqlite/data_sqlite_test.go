// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package sqlite

import (
	"context"
	"maps"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugin/plugintest"
)

func TestSqliteDataSchema(t *testing.T) {
	source := makeSqliteDataSource()
	assert.NotNil(t, source.Config)
	assert.NotNil(t, source.Args)
	assert.NotNil(t, source.DataFunc)
}

func TestSqliteDataCall(t *testing.T) {
	type result struct {
		data  plugindata.Data
		diags diagtest.Asserts
	}
	tt := []struct {
		name     string
		cfg      map[string]cty.Value
		args     map[string]cty.Value
		before   func(tb testing.TB, fs testFS) string
		expected result
		canceled bool
	}{
		{
			name: "empty_database_uri",
			cfg: map[string]cty.Value{
				"database_uri": cty.StringVal(""),
			},
			args: map[string]cty.Value{
				"sql_query": cty.StringVal("SELECT * FROM testdata"),
			},
			expected: result{
				diags: diagtest.Asserts{
					{
						diagtest.IsError,
						diagtest.SummaryContains("non-empty"),
						diagtest.DetailContains("database_uri"),
					},
				},
			},
		},
		{
			name: "nil_database_uri",
			cfg: map[string]cty.Value{
				"database_uri": cty.NullVal(cty.String),
			},
			expected: result{
				diags: diagtest.Asserts{
					{
						diagtest.IsError,
						diagtest.DetailContains("sql_query", "required"),
					},
					{
						diagtest.IsError,
						diagtest.DetailContains("database_uri", "null"),
					},
				},
			},
		},
		{
			name: "empty_sql_query",
			cfg: map[string]cty.Value{
				"database_uri": cty.StringVal("file:./file.db"),
			},
			args: (map[string]cty.Value{
				"sql_query": cty.StringVal(""),
			}),
			expected: result{
				diags: diagtest.Asserts{
					{
						diagtest.IsError,
						diagtest.SummaryEquals("Invalid arguments"),
						diagtest.DetailEquals("sql_query is required"),
					},
				},
			},
		},
		{
			name: "nil_sql_query",
			cfg: (map[string]cty.Value{
				"database_uri": cty.StringVal("file:./file.db"),
			}),
			args: (map[string]cty.Value{
				"sql_query": cty.NullVal(cty.String),
			}),
			expected: result{
				diags: diagtest.Asserts{
					{
						diagtest.IsError,
						diagtest.SummaryContains("non-null"),
						diagtest.DetailContains("sql_query"),
					},
				},
			},
		},
		{
			name: "empty_table",
			args: (map[string]cty.Value{
				"sql_query": cty.StringVal("SELECT * FROM testdata"),
				"sql_args":  cty.ListValEmpty(cty.DynamicPseudoType),
			}),
			before: func(tb testing.TB, fs testFS) string {
				dsn := "file:" + filepath.Join(fs.path, "file.db")
				prepareTestDB(tb, testData{
					dsn:    dsn,
					schema: "CREATE TABLE testdata (id INTEGER PRIMARY KEY, text_val TEXT)",
					data:   []map[string]any{},
				})
				return dsn
			},
			expected: result{
				data: plugindata.List{},
			},
		},
		{
			name: "non_empty_table",
			args: (map[string]cty.Value{
				"sql_query": cty.StringVal("SELECT * FROM testdata"),
				"sql_args":  cty.ListValEmpty(cty.DynamicPseudoType),
			}),
			before: func(tb testing.TB, fs testFS) string {
				dsn := "file:" + filepath.Join(fs.path, "file.db")
				prepareTestDB(tb, testData{
					dsn:    dsn,
					schema: "CREATE TABLE testdata (id INTEGER PRIMARY KEY, text_val TEXT, num_val INTEGER, bool_val BOOLEAN)",
					data: []map[string]any{
						{
							"id":       int64(1),
							"text_val": "text_1",
							"num_val":  int64(1),
							"bool_val": true,
						},
						{
							"id":       int64(2),
							"text_val": "text_2",
							"num_val":  int64(2),
							"bool_val": false,
						},
					},
				})
				return dsn
			},
			expected: result{
				data: plugindata.List{
					plugindata.Map{
						"id":       plugindata.Number(1),
						"text_val": plugindata.String("text_1"),
						"num_val":  plugindata.Number(1),
						"bool_val": plugindata.Bool(true),
					},
					plugindata.Map{
						"id":       plugindata.Number(2),
						"text_val": plugindata.String("text_2"),
						"num_val":  plugindata.Number(2),
						"bool_val": plugindata.Bool(false),
					},
				},
			},
		},
		{
			name: "with_sql_args",
			args: (map[string]cty.Value{
				"sql_query": cty.StringVal("SELECT * FROM testdata WHERE text_val = $1 AND num_val = $2 AND bool_val = $3 AND null_val IS $4;"),
				"sql_args": cty.TupleVal([]cty.Value{
					cty.StringVal("text_2"),
					cty.NumberIntVal(2),
					cty.BoolVal(false),
					cty.NullVal(cty.DynamicPseudoType),
				}),
			}),
			before: func(tb testing.TB, fs testFS) string {
				dsn := "file:" + filepath.Join(fs.path, "file.db")
				prepareTestDB(tb, testData{
					dsn:    dsn,
					schema: "CREATE TABLE testdata (id INTEGER PRIMARY KEY, text_val TEXT, num_val INTEGER, bool_val BOOLEAN, null_val TEXT DEFAULT NULL)",
					data: []map[string]any{
						{
							"id":       int64(1),
							"text_val": "text_1",
							"num_val":  int64(1),
							"bool_val": true,
						},
						{
							"id":       int64(2),
							"text_val": "text_2",
							"num_val":  int64(2),
							"bool_val": false,
						},
					},
				})
				return dsn
			},
			expected: result{
				data: plugindata.List{
					plugindata.Map{
						"id":       plugindata.Number(2),
						"text_val": plugindata.String("text_2"),
						"num_val":  plugindata.Number(2),
						"bool_val": plugindata.Bool(false),
						"null_val": nil,
					},
				},
			},
		},
		{
			name: "missing_sql_args",
			args: (map[string]cty.Value{
				"sql_query": cty.StringVal("SELECT * FROM testdata WHERE bool_val = $1;"),
				"sql_args":  cty.ListValEmpty(cty.DynamicPseudoType),
			}),
			before: func(tb testing.TB, fs testFS) string {
				dsn := "file:" + filepath.Join(fs.path, "file.db")
				prepareTestDB(tb, testData{
					dsn:    dsn,
					schema: "CREATE TABLE testdata (id INTEGER PRIMARY KEY, text_val TEXT, num_val INTEGER, bool_val BOOLEAN)",
					data: []map[string]any{
						{
							"id":       int64(1),
							"text_val": "text_1",
							"num_val":  int64(1),
							"bool_val": true,
						},
						{
							"id":       int64(2),
							"text_val": "text_2",
							"num_val":  int64(2),
							"bool_val": false,
						},
					},
				})
				return dsn
			},
			expected: result{
				diags: diagtest.Asserts{
					{
						diagtest.IsError,
						diagtest.SummaryEquals("Failed to query database"),
						diagtest.DetailEquals("not enough args to execute query: want 1 got 0"),
					},
				},
			},
		},
		{
			name: "table_not_found",
			args: (map[string]cty.Value{
				"sql_query": cty.StringVal("SELECT * FROM testdata"),
				"sql_args":  cty.ListValEmpty(cty.DynamicPseudoType),
			}),
			before: func(tb testing.TB, fs testFS) string {
				dsn := "file:" + filepath.Join(fs.path, "file.db")
				prepareTestDB(tb, testData{
					dsn:    dsn,
					schema: "CREATE TABLE testdata_other (id INTEGER PRIMARY KEY)",
					data:   []map[string]any{},
				})
				return dsn
			},
			expected: result{
				diags: diagtest.Asserts{
					{
						diagtest.IsError,
						diagtest.SummaryEquals("Failed to query database"),
						diagtest.DetailEquals("no such table: testdata"),
					},
				},
			},
		},
		{
			name: "canceled",
			cfg: (map[string]cty.Value{
				"database_uri": cty.StringVal("file:./file.db"),
			}),
			args: (map[string]cty.Value{
				"sql_query": cty.StringVal("SELECT * FROM testdata"),
				"sql_args":  cty.ListValEmpty(cty.DynamicPseudoType),
			}),
			before: func(tb testing.TB, fs testFS) string {
				dsn := "file:" + filepath.Join(fs.path, "file.db")
				prepareTestDB(tb, testData{
					dsn:    dsn,
					schema: "CREATE TABLE testdata (id INTEGER PRIMARY KEY, text_val TEXT, num_val INTEGER, bool_val BOOLEAN)",
					data: []map[string]any{
						{
							"id":       int64(1),
							"text_val": "text_1",
							"num_val":  int64(1),
							"bool_val": true,
						},
						{
							"id":       int64(2),
							"text_val": "text_2",
							"num_val":  int64(2),
							"bool_val": false,
						},
					},
				})
				return dsn
			},
			expected: result{
				diags: diagtest.Asserts{{
					diagtest.IsError,
					diagtest.SummaryEquals("Failed to query database"),
					diagtest.DetailEquals("context canceled"),
				}},
			},
			canceled: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if runtime.GOOS == "windows" && tc.name == "missing_sql_args" {
				t.Skip("Skipping missing_sql_args test: bug in sqlite prevents deleting the db")
			}

			p := Plugin("1.2.3")
			config := plugintest.NewTestDecoder(t, p.DataSources["sqlite"].Config).
				SetHeaders("config", "data", "sqlite")
			args := plugintest.NewTestDecoder(t, p.DataSources["sqlite"].Args).
				SetHeaders("data", "sqlite", `"test"`)

			for k, v := range tc.args {
				args.SetAttr(k, v)
			}
			for k, v := range tc.cfg {
				config.SetAttr(k, v)
			}

			if tc.before != nil {
				fs := makeTestFS(t)
				dsn := tc.before(t, fs)
				config.SetAttr("database_uri", cty.StringVal(dsn))
			}
			params := plugin.RetrieveDataParams{}

			var diags, diag diagnostics.Diag
			var fm, fm2 map[string]*hcl.File
			params.Args, fm, diags = args.DecodeDiagFiles()
			params.Config, fm2, diag = config.DecodeDiagFiles()
			diags.Extend(diag)
			maps.Copy(fm, fm2)
			var data plugindata.Data

			if !diags.HasErrors() {
				ctx := context.Background()
				if tc.canceled {
					nextCtx, cancel := context.WithCancel(ctx)
					cancel()
					ctx = nextCtx
				}
				data, diag = p.RetrieveData(ctx, "sqlite", &params)
				diags.Extend(diag)
			}
			tc.expected.diags.AssertMatch(t, diags, fm)
			assert.Equal(t, tc.expected.data, data, "unexpected result")
		})
	}
}
