// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package iris

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugins/iris/client"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeIrisAlertsDataSource(loader ClientLoadFn) *plugin.DataSource {
	return &plugin.DataSource{
		DataFunc: fetchIrisAlertsData(loader),
		Doc:      "Retrieves alerts from DFIR-IRIS. Supports filtering by alert, case, customer, owner, severity, classification, state, source, tags, and date range.",
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "api_url",
					Type:        cty.String,
					Constraints: constraint.RequiredMeaningful,
					Doc:         "Iris API url",
				},
				{
					Name:        "api_key",
					Type:        cty.String,
					Secret:      true,
					Constraints: constraint.RequiredMeaningful,
					Doc:         "Iris API Key",
				},
				{
					Name:       "insecure",
					Type:       cty.Bool,
					DefaultVal: cty.BoolVal(false),
					Doc:        "Enable/disable insecure TLS",
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "alert_ids",
					Type: cty.List(cty.Number),
					Doc:  "List of Alert IDs",
				},
				{
					Name: "alert_source",
					Type: cty.String,
					Doc:  "Alert Source",
				},
				{
					Name: "tags",
					Type: cty.List(cty.String),
					Doc:  "List of tags",
				},
				{
					Name: "case_id",
					Type: cty.Number,
					Doc:  "Case ID",
				},
				{
					Name: "customer_id",
					Type: cty.Number,
					Doc:  "Alert Customer ID",
				},
				{
					Name: "owner_id",
					Type: cty.Number,
					Doc:  "Alert Owner ID",
				},
				{
					Name: "severity_id",
					Type: cty.Number,
					Doc:  "Alert Severity ID",
				},
				{
					Name: "classification_id",
					Type: cty.Number,
					Doc:  "Alert Classification ID",
				},
				{
					Name: "status_id",
					Type: cty.Number,
					Doc:  "Alert State ID",
				},
				{
					Name: "alert_start_date",
					Type: cty.String,
					Doc:  "Alert Date - lower boundary",
				},
				{
					Name: "alert_end_date",
					Type: cty.String,
					Doc:  "Alert Date - higher boundary",
				},
				{
					Name:       "sort",
					Type:       cty.String,
					Doc:        "Sort order",
					DefaultVal: cty.StringVal("desc"),
					OneOf: []cty.Value{
						cty.StringVal("desc"),
						cty.StringVal("asc"),
					},
				},
				{
					Name:         "size",
					Type:         cty.Number,
					Doc:          "Size limit to retrieve",
					MinInclusive: cty.NumberIntVal(0),
					Constraints:  constraint.NonNull,
					DefaultVal:   cty.NumberIntVal(0),
				},
			},
		},
	}
}

func fetchIrisAlertsData(loader ClientLoadFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		cli, err := parseConfig(params.Config, loader)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse configuration",
			}}
		}
		req, err := parseListAlertsReq(params.Args)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse arguments",
			}}
		}
		num, _ := params.Args.GetAttrVal("size").AsBigFloat().Int64()
		size := int(num)
		req.Page = 1
		var cases plugindata.List
		for {
			res, err := cli.ListAlerts(ctx, req)
			if err != nil {
				return nil, handleClientError(err)
			}
			if res.Data == nil {
				break
			}
			for _, v := range res.Data.Alerts {
				data, err := plugindata.ParseAny(v)
				if err != nil {
					return nil, diagnostics.Diag{{
						Severity: hcl.DiagError,
						Summary:  "Failed to parse data",
					}}
				}
				cases = append(cases, data)
				if size > 0 && len(cases) == size {
					break
				}
			}
			if (size > 0 && len(cases) == size) || res.Data.NextPage == nil {
				break
			}
			req.Page = *res.Data.NextPage
		}
		return cases, nil
	}
}

func parseListAlertsReq(args *dataspec.Block) (*client.ListAlertsReq, error) {
	if args == nil {
		return nil, fmt.Errorf("arguments are required")
	}
	req := &client.ListAlertsReq{}
	if attr := args.GetAttrVal("alert_ids"); !attr.IsNull() {
		ids := attr.AsValueSlice()
		for _, id := range ids {
			if id.IsNull() {
				continue
			}
			num, _ := id.AsBigFloat().Int64()
			req.AlertIDs = append(req.AlertIDs, int(num))
		}
	}
	if attr := args.GetAttrVal("tags"); !attr.IsNull() {
		tags := attr.AsValueSlice()
		for _, tag := range tags {
			if tag.IsNull() {
				continue
			}
			req.AlertTags = append(req.AlertTags, tag.AsString())
		}
	}
	if attr := args.GetAttrVal("case_id"); !attr.IsNull() {
		num, _ := attr.AsBigFloat().Int64()
		req.CaseID = client.Int(int(num))
	}
	if attr := args.GetAttrVal("classification_id"); !attr.IsNull() {
		num, _ := attr.AsBigFloat().Int64()
		req.AlertClassificationID = client.Int(int(num))
	}
	if attr := args.GetAttrVal("customer_id"); !attr.IsNull() {
		num, _ := attr.AsBigFloat().Int64()
		req.AlertCustomerID = client.Int(int(num))
	}
	if attr := args.GetAttrVal("owner_id"); !attr.IsNull() {
		num, _ := attr.AsBigFloat().Int64()
		req.AlertOwnerID = client.Int(int(num))
	}
	if attr := args.GetAttrVal("severity_id"); !attr.IsNull() {
		num, _ := attr.AsBigFloat().Int64()
		req.AlertSeverityID = client.Int(int(num))
	}
	if attr := args.GetAttrVal("classification_id"); !attr.IsNull() {
		num, _ := attr.AsBigFloat().Int64()
		req.AlertClassificationID = client.Int(int(num))
	}
	if attr := args.GetAttrVal("status_id"); !attr.IsNull() {
		num, _ := attr.AsBigFloat().Int64()
		req.AlertStatusID = client.Int(int(num))
	}
	if attr := args.GetAttrVal("alert_source"); !attr.IsNull() {
		req.AlertSource = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("alert_start_date"); !attr.IsNull() {
		req.AlertStartDate = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("alert_end_date"); !attr.IsNull() {
		req.AlertEndDate = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("sort"); !attr.IsNull() {
		req.Sort = client.String(attr.AsString())
	}
	return req, nil
}
