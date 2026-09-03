// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package elastic

import (
	"context"
	"log/slog"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

const maxSimpleSearchResultsSize = 10000

func makeElasticSearchDataSource() *plugin.DataSource {
	return &plugin.DataSource{
		Doc:      "Retrieves documents and aggregations from Elasticsearch by document ID, query string, or Query DSL query.",
		DataFunc: fetchElasticSearchData,
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "base_url",
					Type: cty.String,
				},
				{
					Name: "cloud_id",
					Type: cty.String,
				},
				{
					Name:   "api_key_str",
					Type:   cty.String,
					Secret: true,
				},
				{
					Name:   "api_key",
					Type:   cty.List(cty.String),
					Secret: true,
				},
				{
					Name: "basic_auth_username",
					Type: cty.String,
				},
				{
					Name:   "basic_auth_password",
					Type:   cty.String,
					Secret: true,
				},
				{
					Name:   "bearer_auth",
					Type:   cty.String,
					Secret: true,
				},
				{
					Name:   "ca_certs",
					Type:   cty.String,
					Secret: true,
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "index",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
				{
					Name:        "id",
					Type:        cty.String,
					Constraints: constraint.NonNull,
				},
				{
					Name: "query_string",
					Type: cty.String,
				},
				{
					Name: "query",
					Type: cty.Map(cty.DynamicPseudoType),
				},
				{
					Name: "aggs",
					Type: cty.DynamicPseudoType,
				},
				{
					Name: "only_hits",
					Type: cty.Bool,
				},
				{
					Name: "fields",
					Type: cty.List(cty.String),
				},
				{
					Name:         "size",
					Type:         cty.Number,
					Constraints:  constraint.NonNull,
					DefaultVal:   cty.NumberIntVal(1000),
					MinInclusive: cty.NumberIntVal(0),
				},
			},
		},
	}
}

func fetchElasticSearchData(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
	client, err := makeSearchClient(params.Config)
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to create elasticsearch client",
			Detail:   err.Error(),
		}}
	}
	var diags diagnostics.Diag
	if (params.Args.GetAttrVal("only_hits").IsNull() || params.Args.GetAttrVal("only_hits").True()) &&
		!params.Args.GetAttrVal("aggs").IsNull() {
		if params.Args.GetAttrVal("query").IsNull() && params.Args.GetAttrVal("query_string").IsNull() {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Invalid arguments",
				Detail:   "Aggregations are not supported without a query or query_string",
			}}
		}
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Aggregations are not supported",
			Detail:   "Aggregations are not supported when only_hits is true",
		})
	}

	index := params.Args.GetAttrVal("index")
	if index.IsNull() {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Invalid arguments",
			Detail:   "Index value is required",
		}}
	}

	var data plugindata.Data
	if params.Args.HasAttr("id") {
		data, err = getByID(client.Get, params.Args)
	} else {
		size64, _ := params.Args.GetAttrVal("size").AsBigFloat().Int64()
		size := int(size64)
		if size <= maxSimpleSearchResultsSize {
			slog.DebugContext(ctx, "Sending normal search request", "size", size)
			data, err = search(client.Search, params.Args, size)
		} else {
			slog.DebugContext(ctx, "Starting a scroll search request", "size", size)
			data, err = searchWithScroll(client, params.Args, size)
		}
	}
	if err != nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to fetch data",
			Detail:   err.Error(),
		})
	} else {
		slog.DebugContext(ctx, "Returning data received from Elasticsearch")
	}
	return data, diags
}
