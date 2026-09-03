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

func makeIrisCasesDataSource(loader ClientLoadFn) *plugin.DataSource {
	return &plugin.DataSource{
		Doc:      "Retrieves cases from DFIR-IRIS. Supports filtering by case, customer, owner, severity, state, SOC, and opening date.",
		DataFunc: fetchIrisCasesData(loader),
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
					Name: "case_ids",
					Type: cty.List(cty.Number),
					Doc:  "List of Case IDs",
				},
				{
					Name: "customer_id",
					Type: cty.Number,
					Doc:  "Case Customer ID",
				},
				{
					Name: "owner_id",
					Type: cty.Number,
					Doc:  "Case Owner ID",
				},
				{
					Name: "severity_id",
					Type: cty.Number,
					Doc:  "Case Severity ID",
				},
				{
					Name: "state_id",
					Type: cty.Number,
					Doc:  "Case State ID",
				},
				{
					Name: "soc_id",
					Type: cty.String,
					Doc:  "Case SOC ID",
				},
				{
					Name: "start_open_date",
					Type: cty.String,
					Doc:  "Case opening date - lower boundary",
				},
				{
					Name: "end_open_date",
					Type: cty.String,
					Doc:  "Case opening date - higher boundary",
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

func fetchIrisCasesData(loader ClientLoadFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		cli, err := parseConfig(params.Config, loader)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse configuration",
			}}
		}
		req, err := parseListCasesReq(params.Args)
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
			res, err := cli.ListCases(ctx, req)
			if err != nil {
				return nil, handleClientError(err)
			}
			if res.Data == nil {
				break
			}
			for _, v := range res.Data.Cases {
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

func parseListCasesReq(args *dataspec.Block) (*client.ListCasesReq, error) {
	if args == nil {
		return nil, fmt.Errorf("arguments are required")
	}
	req := &client.ListCasesReq{}
	if attr := args.GetAttrVal("case_ids"); !attr.IsNull() {
		ids := attr.AsValueSlice()
		for _, id := range ids {
			if id.IsNull() {
				continue
			}
			num, _ := id.AsBigFloat().Int64()
			req.CaseIDs = append(req.CaseIDs, int(num))
		}
	}
	if attr := args.GetAttrVal("customer_id"); !attr.IsNull() {
		num, _ := attr.AsBigFloat().Int64()
		req.CaseCustomerID = client.Int(int(num))
	}
	if attr := args.GetAttrVal("owner_id"); !attr.IsNull() {
		num, _ := attr.AsBigFloat().Int64()
		req.CaseOwnerID = client.Int(int(num))
	}
	if attr := args.GetAttrVal("severity_id"); !attr.IsNull() {
		num, _ := attr.AsBigFloat().Int64()
		req.CaseSeverityID = client.Int(int(num))
	}
	if attr := args.GetAttrVal("state_id"); !attr.IsNull() {
		num, _ := attr.AsBigFloat().Int64()
		req.CaseStateID = client.Int(int(num))
	}
	if attr := args.GetAttrVal("soc_id"); !attr.IsNull() {
		req.CaseSocID = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("start_open_date"); !attr.IsNull() {
		req.StartOpenDate = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("end_open_date"); !attr.IsNull() {
		req.EndOpenDate = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("sort"); !attr.IsNull() {
		req.Sort = client.String(attr.AsString())
	}
	return req, nil
}
