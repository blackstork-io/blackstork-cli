// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package microsoft

import (
	"context"
	"net/url"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeMicrosoftSecurityDataSource(loader MicrosoftSecurityClientLoadFn) *plugin.DataSource {
	return &plugin.DataSource{
		Doc:      "Sends a GET request to a Microsoft Security API endpoint and returns one object or a paginated list of objects.",
		DataFunc: fetchMicrosoftSecurity(loader),
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Doc:         "The Azure client ID",
					Name:        "client_id",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
				{
					Doc:    "The Azure client secret. Required if `private_key_file` or `private_key` is not provided.",
					Name:   "client_secret",
					Type:   cty.String,
					Secret: true,
				},
				{
					Doc:         "The Azure tenant ID",
					Name:        "tenant_id",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
				{
					Doc:  "The path to the private key file. Ignored if `private_key` or `client_secret` is provided.",
					Name: "private_key_file",
					Type: cty.String,
				},
				{
					Doc:  "The private key contents. Ignored if `client_secret` is provided.",
					Name: "private_key",
					Type: cty.String,
				},
				{
					Doc:  "The key passphrase. Ignored if `client_secret` is provided.",
					Name: "key_passphrase",
					Type: cty.String,
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "endpoint",
					Doc:         "API endpoint to query",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
					ExampleVal:  cty.StringVal("/users"),
				},
				{
					Name: "query_params",
					Doc:  "HTTP query parameters",
					Type: cty.Map(cty.String),
				},
				{
					Name:         "size",
					Doc:          "Number of objects to be returned",
					Type:         cty.Number,
					Constraints:  constraint.NonNull,
					DefaultVal:   cty.NumberIntVal(50),
					MinInclusive: cty.NumberIntVal(1),
				},
				{
					Name:       "is_object_endpoint",
					Doc:        "Indicates if API endpoint serves a single object. If set to `true`, `query_params` and `size` arguments are ignored.",
					Type:       cty.Bool,
					DefaultVal: cty.BoolVal(false),
				},
			},
		},
	}
}

func fetchMicrosoftSecurity(loader MicrosoftSecurityClientLoadFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		cli, err := loader(ctx, params.Config)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Unable to create Microsoft Security API client",
				Detail:   err.Error(),
			}}
		}
		endPoint := params.Args.GetAttrVal("endpoint").AsString()
		isObjectEndpoint := params.Args.GetAttrVal("is_object_endpoint")

		queryParamsAttr := params.Args.GetAttrVal("query_params")
		var queryParams url.Values

		if !queryParamsAttr.IsNull() {
			queryParams = url.Values{}
			queryMap := queryParamsAttr.AsValueMap()
			for k, v := range queryMap {
				queryParams.Add(k, v.AsString())
			}
		}

		var response plugindata.Data

		if isObjectEndpoint.True() {
			response, err = cli.QueryObject(ctx, endPoint, queryParams)
		} else {
			size64, _ := params.Args.GetAttrVal("size").AsBigFloat().Int64()
			size := int(size64)
			response, err = cli.QueryObjects(ctx, endPoint, queryParams, size)
		}
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to query Microsoft Security API",
				Detail:   err.Error(),
			}}
		}
		return response, nil
	}
}
