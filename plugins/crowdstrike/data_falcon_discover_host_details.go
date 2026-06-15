// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package crowdstrike

import (
	"context"

	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client/discover"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeFalconDiscoverHostDetailsDataSource(loader ClientLoaderFn) *plugin.DataSource {
	return &plugin.DataSource{
		Doc:      "The `falcon_discover_host_details` data source fetches host details from Falcon Discover Host API.",
		DataFunc: fetchFalconDiscoverHostDetails(loader),
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
					Doc:  "Host search expression using Falcon Query Language (FQL)",
				},
			},
		},
	}
}

func fetchFalconDiscoverHostDetails(loader ClientLoaderFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		cli, err := loader(makeApiConfig(ctx, params.Config))
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Unable to create falcon client",
				Detail:   err.Error(),
			}}
		}
		limit, _ := params.Args.GetAttrVal("limit").AsBigFloat().Int64()
		queryHostParams := discover.NewQueryHostsParams().WithDefaults()
		queryHostParams.SetLimit(&limit)
		queryHostParams.SetContext(ctx)
		queryHostsResponse, err := cli.Discover().QueryHosts(queryHostParams)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to query Falcon Discover Hosts",
				Detail:   err.Error(),
			}}
		}
		if err = falcon.AssertNoError(queryHostsResponse.GetPayload().Errors); err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to query Falcon Discover Hosts",
				Detail:   err.Error(),
			}}
		}
		hostIds := queryHostsResponse.GetPayload().Resources

		getHostParams := discover.NewGetHostsParams().WithDefaults()
		getHostParams.SetIds(hostIds)
		getHostParams.SetContext(ctx)
		getHostsResponse, err := cli.Discover().GetHosts(getHostParams)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to fetch Falcon Discover Hosts",
				Detail:   err.Error(),
			}}
		}
		if err = falcon.AssertNoError(queryHostsResponse.GetPayload().Errors); err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to fetch Falcon Discover Hosts",
				Detail:   err.Error(),
			}}
		}

		resources := getHostsResponse.GetPayload().Resources
		data, err := encodeResponse(resources)
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
