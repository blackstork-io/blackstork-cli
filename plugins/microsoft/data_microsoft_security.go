package microsoft

import (
	"context"
	"net/url"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

func makeMicrosoftSecurityDataSource(loader MicrosoftSecurityClientLoadFn) *plugin.DataSource {
	return &plugin.DataSource{
		Doc:      "The `microsoft_security` data source queries Microsoft Security API.",
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
