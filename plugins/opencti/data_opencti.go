package opencti

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
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
