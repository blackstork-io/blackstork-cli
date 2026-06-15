// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package snyk

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugins/snyk/client"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeSnykIssuesDataSource(loader ClientLoadFn) *plugin.DataSource {
	return &plugin.DataSource{
		Doc:      "The `snyk_issues` data source fetches issues from Snyk.",
		DataFunc: fetchSnykIssues(loader),
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Doc:         "The Snyk API key",
					Name:        "api_key",
					Type:        cty.String,
					Constraints: constraint.RequiredMeaningful,
					Secret:      true,
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Doc:  "The group ID",
					Name: "group_id",
					Type: cty.String,
				},
				{
					Doc:  "The organization ID",
					Name: "org_id",
					Type: cty.String,
				},
				{
					Doc:  "The scan item ID",
					Name: "scan_item_id",
					Type: cty.String,
				},
				{
					Doc:  "The scan item type",
					Name: "scan_item_type",
					Type: cty.String,
				},
				{
					Doc:  "The issue type",
					Name: "type",
					Type: cty.String,
				},
				{
					Doc:  "The updated before date",
					Name: "updated_before",
					Type: cty.String,
				},
				{
					Doc:  "The updated after date",
					Name: "updated_after",
					Type: cty.String,
				},
				{
					Doc:  "The created before date",
					Name: "created_before",
					Type: cty.String,
				},
				{
					Doc:  "The created after date",
					Name: "created_after",
					Type: cty.String,
				},
				{
					Doc:  "The effective severity level",
					Name: "effective_severity_level",
					Type: cty.List(cty.String),
				},
				{
					Doc:  "The status",
					Name: "status",
					Type: cty.List(cty.String),
				},
				{
					Doc:  "The ignored flag",
					Name: "ignored",
					Type: cty.Bool,
				},
				{
					Doc:          "The limit of issues to fetch",
					Name:         "limit",
					Type:         cty.Number,
					Constraints:  constraint.NonNull,
					MinInclusive: cty.NumberIntVal(0),
					DefaultVal:   cty.NumberIntVal(0),
				},
			},
		},
	}
}

func fetchSnykIssues(loader ClientLoadFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		client := loader(params.Config.GetAttrVal("api_key").AsString())
		limit, diags := params.Args.Attrs["limit"].GetInt()
		if diags.HasErrors() {
			return nil, diags
		}
		req, err := makeRequest(params.Args)
		if err != nil {
			return nil, diagnostics.Diag{
				{
					Severity: hcl.DiagError,
					Summary:  "Failed to create Snyk request",
					Detail:   err.Error(),
				},
			}
		}
		var data plugindata.List
		for {
			res, err := client.ListIssues(ctx, req)
			if err != nil {
				return nil, diagnostics.Diag{
					{
						Severity: hcl.DiagError,
						Summary:  "Failed to list Snyk issues",
						Detail:   err.Error(),
					},
				}
			}
			for _, v := range res.Data {
				item, err := plugindata.ParseAny(v)
				if err != nil {
					return nil, diagnostics.Diag{
						{
							Severity: hcl.DiagError,
							Summary:  "Failed to parse Snyk issue",
							Detail:   err.Error(),
						},
					}
				}
				data = append(data, item)
			}
			if limit > 0 && len(data) >= limit {
				data = data[:limit]
				break
			}
			if links := res.Links; links != nil && links.Next != nil {
				req.StartingAfter = links.Next
			} else {
				break
			}
		}
		return data, nil
	}
}

func makeRequest(args *dataspec.Block) (*client.ListIssuesReq, error) {
	req := &client.ListIssuesReq{
		Limit: pageSize,
	}
	groupID := args.GetAttrVal("group_id")
	orgID := args.GetAttrVal("org_id")
	if groupID.IsNull() && orgID.IsNull() {
		return nil, fmt.Errorf("either group_id or org_id must be set")
	}
	if !groupID.IsNull() && !orgID.IsNull() {
		return nil, fmt.Errorf("only one of group_id or org_id is allowed")
	}
	if !groupID.IsNull() {
		req.GroupID = client.String(groupID.AsString())
	}
	if !orgID.IsNull() {
		req.OrgID = client.String(orgID.AsString())
	}
	if arg := args.GetAttrVal("scan_item_id"); !arg.IsNull() {
		req.ScanItemID = client.String(arg.AsString())
	}
	if arg := args.GetAttrVal("scan_item_type"); !arg.IsNull() {
		req.ScanItemType = client.String(arg.AsString())
	}
	if arg := args.GetAttrVal("type"); !arg.IsNull() {
		req.Type = client.String(arg.AsString())
	}
	if arg := args.GetAttrVal("updated_before"); !arg.IsNull() {
		req.UpdatedBefore = client.String(arg.AsString())
	}
	if arg := args.GetAttrVal("updated_after"); !arg.IsNull() {
		req.UpdatedAfter = client.String(arg.AsString())
	}
	if arg := args.GetAttrVal("created_before"); !arg.IsNull() {
		req.CreatedBefore = client.String(arg.AsString())
	}
	if arg := args.GetAttrVal("created_after"); !arg.IsNull() {
		req.CreatedAfter = client.String(arg.AsString())
	}
	if arg := args.GetAttrVal("effective_severity_level"); !arg.IsNull() {
		list := []string{}
		for _, v := range arg.AsValueSlice() {
			list = append(list, v.AsString())
		}
		req.EffectiveSeverityLevel = client.StringList(list)
	}
	if arg := args.GetAttrVal("status"); !arg.IsNull() {
		list := []string{}
		for _, v := range arg.AsValueSlice() {
			list = append(list, v.AsString())
		}
		req.Status = client.StringList(list)
	}
	if arg := args.GetAttrVal("ignored"); !arg.IsNull() {
		req.Ignored = client.Bool(arg.True())
	}

	return req, nil
}
