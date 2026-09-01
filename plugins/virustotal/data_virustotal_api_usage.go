// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

// Package virustotal implements the BlackStork VirusTotal plugin.
package virustotal

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugins/virustotal/client"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeVirusTotalAPIUsageDataSchema(loader ClientLoadFn) *plugin.DataSource {
	return &plugin.DataSource{
		DataFunc: fetchVirusTotalAPIUsageData(loader),
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
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
					Name: "user_id",
					Type: cty.String,
				},
				{
					Name: "group_id",
					Type: cty.String,
				},
				{
					Name: "start_date",
					Type: cty.String,
				},
				{
					Name: "end_date",
					Type: cty.String,
				},
			},
		},
	}
}

func fetchVirusTotalAPIUsageData(loader ClientLoadFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		cli := loader(params.Config.GetAttrVal("api_key").AsString())
		args, err := parseAPIUsageArgs(params.Args)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse arguments",
				Detail:   err.Error(),
			}}
		}
		var data map[string]any
		if args.User != nil {
			req := &client.GetUserAPIUsageReq{
				User: *args.User,
			}
			if args.StartDate != nil {
				req.StartDate = &client.Date{Time: *args.StartDate}
			}
			if args.EndDate != nil {
				req.EndDate = &client.Date{Time: *args.EndDate}
			}

			res, err := cli.GetUserAPIUsage(ctx, req)
			if err != nil {
				return nil, diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Failed to fetch data",
					Detail:   err.Error(),
				}}
			}
			data = res.Data
		} else {
			req := &client.GetGroupAPIUsageReq{
				Group: *args.Group,
			}
			if args.StartDate != nil {
				req.StartDate = &client.Date{Time: *args.StartDate}
			}
			if args.EndDate != nil {
				req.EndDate = &client.Date{Time: *args.EndDate}
			}

			res, err := cli.GetGroupAPIUsage(ctx, req)
			if err != nil {
				return nil, diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Failed to fetch data",
					Detail:   err.Error(),
				}}
			}
			data = res.Data
		}
		result, err := plugindata.ParseMapAny(data)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse data",
				Detail:   err.Error(),
			}}
		}
		return result, nil
	}
}

type apiUsageArgs struct {
	User      *string
	Group     *string
	StartDate *time.Time
	EndDate   *time.Time
}

func parseAPIUsageArgs(args *dataspec.Block) (*apiUsageArgs, error) {
	dst := apiUsageArgs{}

	if args == nil {
		return nil, fmt.Errorf("arguments are null")
	}

	if userID := args.GetAttrVal("user_id"); !userID.IsNull() {
		userIDStr := userID.AsString()
		dst.User = &userIDStr
	}
	if groupID := args.GetAttrVal("group_id"); !groupID.IsNull() {
		groupIDStr := groupID.AsString()
		dst.Group = &groupIDStr
	}
	if dst.User == nil && dst.Group == nil {
		return nil, fmt.Errorf("either user_id or group_id must be set")
	}
	if startDate := args.GetAttrVal("start_date"); !startDate.IsNull() {
		startDateStr := startDate.AsString()
		startDate, err := time.Parse("20060102", startDateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse start_date: %w", err)
		}
		dst.StartDate = &startDate
	}
	if endDate := args.GetAttrVal("end_date"); !endDate.IsNull() {
		endDateStr := endDate.AsString()
		endDate, err := time.Parse("20060102", endDateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse end_date: %w", err)
		}
		dst.EndDate = &endDate
	}
	return &dst, nil
}
