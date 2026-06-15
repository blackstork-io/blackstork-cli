// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package opencti

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeOpenCTIDataSource() *plugin.DataSource {
	return &plugin.DataSource{
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "graphql_url",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
				{
					Name:   "auth_token",
					Type:   cty.String,
					Secret: true,
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "graphql_query",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
			},
		},
		DataFunc: fetchOpenCTIData,
	}
}

func fetchOpenCTIData(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
	url := params.Config.GetAttrVal("graphql_url")
	if url.IsNull() || url.AsString() == "" {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse config",
			Detail:   "graphql_url is required",
		}}
	}
	authToken := params.Config.GetAttrVal("auth_token")
	if authToken.IsNull() {
		authToken = cty.StringVal("")
	}
	query := params.Args.GetAttrVal("graphql_query")
	if query.IsNull() || query.AsString() == "" {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse arguments",
			Detail:   "graphql_query is required",
		}}
	}
	if err := validateQuery(query.AsString()); err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Invalid GraphQL query",
			Detail:   err.Error(),
		}}
	}
	result, err := executeQuery(ctx, url.AsString(), query.AsString(), authToken.AsString())
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to execute query",
			Detail:   err.Error(),
		}}
	}
	return result, nil
}
