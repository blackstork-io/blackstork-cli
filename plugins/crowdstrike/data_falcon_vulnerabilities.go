package crowdstrike

import (
	"context"

	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client/spotlight_vulnerabilities"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

func makeFalconVulnerabilitiesDataSource(loader ClientLoaderFn) *plugin.DataSource {
	return &plugin.DataSource{
		Doc:      "The `falcon_vulnerabilities` data source fetches environment vulnerabilities from Falcon Spotlight API.",
		DataFunc: fetchFalconVulnerabilitiesData(loader),
		Config:   makeDataSourceConfig(),
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "limit",
					Type:        cty.Number,
					Constraints: constraint.Integer,
					DefaultVal:  cty.NumberIntVal(10),
					Doc:         "limit the number of queried items",
				},
				{
					Name: "filter",
					Type: cty.String,
					Doc:  "Vulnerability search expression using Falcon Query Language (FQL)",
				},
				{
					Name: "sort",
					Type: cty.String,
					Doc:  "Vulnerability sort expression using Falcon Query Language (FQL)",
				},
			},
		},
	}
}

func fetchFalconVulnerabilitiesData(loader ClientLoaderFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		cli, err := loader(makeApiConfig(ctx, params.Config))
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Unable to create falcon client",
				Detail:   err.Error(),
			}}
		}
		size, _ := params.Args.GetAttrVal("limit").AsBigFloat().Int64()
		apiParams := spotlight_vulnerabilities.NewCombinedQueryVulnerabilitiesParams().WithDefaults()
		apiParams.SetLimit(&size)
		apiParams.SetContext(ctx)
		if filter := params.Args.GetAttrVal("filter"); !filter.IsNull() {
			apiParams.SetFilter(filter.AsString())
		}
		if sort := params.Args.GetAttrVal("sort"); !sort.IsNull() {
			sortStr := sort.AsString()
			apiParams.SetSort(&sortStr)
		}
		response, err := cli.SpotlightVulnerabilities().CombinedQueryVulnerabilities(apiParams)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to fetch Falcon Spotlight vulnerabilities",
				Detail:   err.Error(),
			}}
		}
		if err = falcon.AssertNoError(response.GetPayload().Errors); err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to fetch Falcon Spotlight vulnerabilities",
				Detail:   err.Error(),
			}}
		}
		vulnerabilities := response.GetPayload().Resources
		data, err := encodeResponse(vulnerabilities)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse response",
				Detail:   err.Error(),
			}}
		}
		return data, nil
	}
}
