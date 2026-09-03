// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package postgresql

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"
	_ "github.com/lib/pq"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makePostgreSQLDataSource() *plugin.DataSource {
	return &plugin.DataSource{
		Doc: "Runs a parameterized SQL query against PostgreSQL and returns the result rows as a list of objects keyed by column name.",
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "database_url",
					Type:        cty.String,
					Constraints: constraint.RequiredMeaningful,
					Secret:      true,
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "sql_query",
					Type:        cty.String,
					Constraints: constraint.RequiredMeaningful,
				},
				{
					Name: "sql_args",
					Type: cty.List(cty.DynamicPseudoType),
				},
			},
		},
		DataFunc: fetchSqliteData,
	}
}

func parseSqliteArgs(args *dataspec.Block) (string, []any, error) {
	sqlQuery := args.GetAttrVal("sql_query")
	if sqlQuery.IsNull() || sqlQuery.AsString() == "" {
		return "", nil, fmt.Errorf("sql_query is required")
	}
	sqlArgs := args.GetAttrVal("sql_args")
	if sqlArgs.IsNull() || sqlArgs.LengthInt() == 0 {
		return sqlQuery.AsString(), nil, nil
	}
	argsList := sqlArgs.AsValueSlice()
	argsResult := make([]any, len(argsList))
	for i, arg := range argsList {
		switch {
		case arg.IsNull():
			argsResult[i] = nil
		case arg.Type() == cty.Number:
			n, _ := arg.AsBigFloat().Float64()
			argsResult[i] = n
		case arg.Type() == cty.String:
			argsResult[i] = arg.AsString()
		case arg.Type() == cty.Bool:
			argsResult[i] = arg.True()
		default:
			return "", nil, fmt.Errorf("sql_args must be a list of strings, numbers, or booleans")
		}
	}
	return sqlQuery.AsString(), argsResult, nil
}

func fetchSqliteData(
	ctx context.Context,
	params *plugin.RetrieveDataParams,
) (plugindata.Data, diagnostics.Diag) {
	dbURL := params.Config.GetAttrVal("database_url").AsString()
	sqlQuery, sqlArgs, err := parseSqliteArgs(params.Args)
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Invalid arguments",
			Detail:   err.Error(),
		}}
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to open database",
			Detail:   err.Error(),
		}}
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, sqlQuery, sqlArgs...)
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to query database",
			Detail:   err.Error(),
		}}
	}
	// read columns
	columns, err := rows.Columns()
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to get column names",
			Detail:   err.Error(),
		}}
	}
	result := make(plugindata.List, 0)

	// read rows
	for rows.Next() {
		// create a map of column name to column value
		columnValArr := make([]nullData, len(columns))
		columnPtrArr := make([]any, len(columns))
		for i := range columns {
			columnPtrArr[i] = &columnValArr[i]
		}
		err = rows.Scan(columnPtrArr...)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to scan row",
				Detail:   err.Error(),
			}}
		}
		row := make(plugindata.Map)
		for i, column := range columns {
			if columnValArr[i].valid {
				row[column] = columnValArr[i].data
			} else {
				row[column] = nil
			}
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to read rows",
			Detail:   err.Error(),
		}}
	}
	return result, nil
}

type nullData struct {
	data  plugindata.Data
	valid bool
}

func (n *nullData) Scan(value any) error {
	if value == nil {
		n.valid = false
		return nil
	}
	switch v := value.(type) {
	case []byte:
		n.data = plugindata.String(base64.StdEncoding.EncodeToString(v))
	case string:
		n.data = plugindata.String(v)
	case int64:
		n.data = plugindata.Number(v)
	case float64:
		n.data = plugindata.Number(v)
	case bool:
		n.data = plugindata.Bool(v)
	case time.Time:
		n.data = plugindata.String(v.Format(time.RFC3339))
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
	n.valid = true
	return nil
}
