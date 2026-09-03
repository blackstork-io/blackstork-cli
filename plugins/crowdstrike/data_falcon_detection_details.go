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
	"github.com/crowdstrike/gofalcon/falcon/client/detects"
	"github.com/crowdstrike/gofalcon/falcon/models"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeFalconDetectionDetailsDataSource(loader ClientLoaderFn) *plugin.DataSource {
	return &plugin.DataSource{
		Doc:      "Retrieves detection details from CrowdStrike Falcon, optionally filtered with Falcon Query Language (FQL). Returns a list of detection records.",
		DataFunc: fetchFalconDetectionDetailsData(loader),
		Config:   makeDataSourceConfig(),
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "filter",
					Type: cty.String,
					Doc:  "Host search expression using Falcon Query Language (FQL)",
				},
				{
					Name:        "limit",
					Type:        cty.Number,
					Constraints: constraint.Integer,
					DefaultVal:  cty.NumberIntVal(10),
					Doc:         "limit the number of queried items",
				},
			},
		},
	}
}

func fetchFalconDetectionDetailsData(loader ClientLoaderFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		cli, err := loader(makeAPIConfig(ctx, params.Config))
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Unable to create falcon client",
				Detail:   err.Error(),
			}}
		}

		response, err := fetchDetects(ctx, cli.Detects(), params)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to query Falcon detects",
				Detail:   err.Error(),
			}}
		}
		if err = falcon.AssertNoError(response.GetPayload().Errors); err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to query Falcon detects",
				Detail:   err.Error(),
			}}
		}

		detectIds := response.GetPayload().Resources
		detailResponse, err := fetchDetectsDetails(ctx, cli.Detects(), detectIds)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to fetch Falcon detect details",
				Detail:   err.Error(),
			}}
		}
		if err = falcon.AssertNoError(response.GetPayload().Errors); err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to fetch Falcon detect details",
				Detail:   err.Error(),
			}}
		}

		resources := detailResponse.GetPayload().Resources
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

func fetchDetects(ctx context.Context, cli DetectsClient, params *plugin.RetrieveDataParams) (*detects.QueryDetectsOK, error) {
	limit, _ := params.Args.GetAttrVal("limit").AsBigFloat().Int64()
	apiParams := &detects.QueryDetectsParams{}
	apiParams.SetLimit(&limit)
	apiParams.Context = ctx
	filter := params.Args.GetAttrVal("filter")
	if !filter.IsNull() {
		filterStr := filter.AsString()
		apiParams.SetFilter(&filterStr)
	}
	return cli.QueryDetects(apiParams)
}

func fetchDetectsDetails(ctx context.Context, cli DetectsClient, detectIds []string) (*detects.GetDetectSummariesOK, error) {
	apiParams := &detects.GetDetectSummariesParams{
		Body: &models.MsaIdsRequest{
			Ids: detectIds,
		},
		Context: ctx,
	}
	return cli.GetDetectSummaries(apiParams)
}
