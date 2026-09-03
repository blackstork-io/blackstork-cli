// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package hackerone

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugins/hackerone/client"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeHackerOneReportsDataSchema(loader ClientLoadFn) *plugin.DataSource {
	return &plugin.DataSource{
		Doc: "Retrieves HackerOne reports available to the configured API account. Supports report, program, state, severity, assignee, reporter, and date filters.",
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "api_username",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
				{
					Name:        "api_token",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
					Secret:      true,
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "size",
					Type: cty.Number,
				},
				{
					Name: "page_number",
					Type: cty.Number,
				},
				{
					Name: "sort",
					Type: cty.String,
				},
				{
					Name: "program",
					Type: cty.List(cty.String),
				},
				{
					Name: "inbox_ids",
					Type: cty.List(cty.Number),
				},
				{
					Name: "reporter",
					Type: cty.List(cty.String),
				},
				{
					Name: "assignee",
					Type: cty.List(cty.String),
				},
				{
					Name: "state",
					Type: cty.List(cty.String),
				},
				{
					Name: "id",
					Type: cty.List(cty.Number),
				},
				{
					Name: "weakness_id",
					Type: cty.List(cty.Number),
				},
				{
					Name: "severity",
					Type: cty.List(cty.String),
				},
				{
					Name: "hacker_published",
					Type: cty.Bool,
				},
				{
					Name: "created_at__gt",
					Type: cty.String,
				},
				{
					Name: "created_at__lt",
					Type: cty.String,
				},
				{
					Name: "submitted_at__gt",
					Type: cty.String,
				},
				{
					Name: "submitted_at__lt",
					Type: cty.String,
				},
				{
					Name: "triaged_at__gt",
					Type: cty.String,
				},
				{
					Name: "triaged_at__lt",
					Type: cty.String,
				},
				{
					Name: "triaged_at__null",
					Type: cty.Bool,
				},
				{
					Name: "closed_at__gt",
					Type: cty.String,
				},
				{
					Name: "closed_at__lt",
					Type: cty.String,
				},
				{
					Name: "closed_at__null",
					Type: cty.Bool,
				},
				{
					Name: "disclosed_at__gt",
					Type: cty.String,
				},
				{
					Name: "disclosed_at__lt",
					Type: cty.String,
				},
				{
					Name: "disclosed_at__null",
					Type: cty.Bool,
				},
				{
					Name: "reporter_agreed_on_going_public",
					Type: cty.Bool,
				},
				{
					Name: "bounty_awarded_at__gt",
					Type: cty.String,
				},
				{
					Name: "bounty_awarded_at__lt",
					Type: cty.String,
				},
				{
					Name: "bounty_awarded_at__null",
					Type: cty.Bool,
				},
				{
					Name: "swag_awarded_at__gt",
					Type: cty.String,
				},
				{
					Name: "swag_awarded_at__lt",
					Type: cty.String,
				},
				{
					Name: "swag_awarded_at__null",
					Type: cty.Bool,
				},
				{
					Name: "last_report_activity_at__gt",
					Type: cty.String,
				},
				{
					Name: "last_report_activity_at__lt",
					Type: cty.String,
				},
				{
					Name: "first_program_activity_at__gt",
					Type: cty.String,
				},
				{
					Name: "first_program_activity_at__lt",
					Type: cty.String,
				},
				{
					Name: "first_program_activity_at__null",
					Type: cty.Bool,
				},
				{
					Name: "last_program_activity_at__gt",
					Type: cty.String,
				},
				{
					Name: "last_program_activity_at__lt",
					Type: cty.String,
				},
				{
					Name: "last_program_activity_at__null",
					Type: cty.Bool,
				},
				{
					Name: "last_activity_at__gt",
					Type: cty.String,
				},
				{
					Name: "last_activity_at__lt",
					Type: cty.String,
				},
				{
					Name: "last_public_activity_at__gt",
					Type: cty.String,
				},
				{
					Name: "last_public_activity_at__lt",
					Type: cty.String,
				},
				{
					Name: "keyword",
					Type: cty.String,
				},
				{
					Name: "custom_fields",
					Type: cty.Map(cty.String),
				},
			},
		},
		DataFunc: fetchHackerOneReports(loader),
	}
}

func fetchHackerOneReports(loader ClientLoadFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		cli, err := makeClient(loader, params.Config)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to create client",
				Detail:   err.Error(),
			}}
		}
		req, err := parseHackerOneReportsArgs(params.Args)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse arguments",
				Detail:   err.Error(),
			}}
		}

		data := make([]any, 0)
		if req.PageNumber != nil {
			res, err := cli.GetAllReports(ctx, req)
			if err != nil {
				return nil, diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Failed to fetch reports",
					Detail:   err.Error(),
				}}
			}
			data = append(data, res.Data...)
		} else {
			limit := -1
			if req.PageSize != nil {
				limit = *req.PageSize
			}
			for page := minPage; ; page++ {
				req.PageNumber = client.Int(page)
				res, err := cli.GetAllReports(ctx, req)
				if err != nil {
					return nil, diagnostics.Diag{{
						Severity: hcl.DiagError,
						Summary:  "Failed to fetch reports",
						Detail:   err.Error(),
					}}
				}
				if res.Data == nil {
					res.Data = make([]any, 0)
				}
				data = append(data, res.Data...)
				if len(res.Data) == 0 || (limit > 0 && len(data) >= limit) {
					break
				}
			}
		}
		dst, err := plugindata.ParseAny(data)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse data",
				Detail:   err.Error(),
			}}
		}
		return dst, nil
	}
}

func parseHackerOneReportsArgs(args *dataspec.Block) (*client.GetAllReportsReq, error) {
	if args == nil {
		return nil, fmt.Errorf("args are required")
	}
	var req client.GetAllReportsReq
	size := args.GetAttrVal("size")
	if !size.IsNull() {
		n, _ := size.AsBigFloat().Int64()
		if n <= 0 {
			return nil, fmt.Errorf("size must be greater than 0")
		}
		req.PageSize = client.Int(int(n))
	}
	pageNumber := args.GetAttrVal("page_number")
	if !pageNumber.IsNull() {
		n, _ := pageNumber.AsBigFloat().Int64()
		if n <= 0 {
			return nil, fmt.Errorf("page_number must be greater than 0")
		}
		req.PageNumber = client.Int(int(n))
	}
	sort := args.GetAttrVal("sort")
	if !sort.IsNull() && sort.AsString() != "" {
		req.Sort = client.String(sort.AsString())
	}
	program := args.GetAttrVal("program")
	if !program.IsNull() {
		programs := program.AsValueSlice()
		for _, p := range programs {
			req.FilterProgram = append(req.FilterProgram, p.AsString())
		}
	}
	inboxIDs := args.GetAttrVal("inbox_ids")
	if !inboxIDs.IsNull() {
		ids := inboxIDs.AsValueSlice()
		for _, id := range ids {
			n, _ := id.AsBigFloat().Int64()
			req.FilterInboxIDs = append(req.FilterInboxIDs, int(n))
		}
	}
	if len(req.FilterProgram)+len(req.FilterInboxIDs) == 0 {
		return nil, fmt.Errorf("at least one of program or inbox_ids must be provided")
	}
	reporter := args.GetAttrVal("reporter")
	if !reporter.IsNull() {
		reporters := reporter.AsValueSlice()
		for _, r := range reporters {
			req.FilterReporter = append(req.FilterReporter, r.AsString())
		}
	}
	assignee := args.GetAttrVal("assignee")
	if !assignee.IsNull() {
		assignees := assignee.AsValueSlice()
		for _, a := range assignees {
			req.FilterAssignee = append(req.FilterAssignee, a.AsString())
		}
	}
	state := args.GetAttrVal("state")
	if !state.IsNull() {
		states := state.AsValueSlice()
		for _, s := range states {
			req.FilterState = append(req.FilterState, s.AsString())
		}
	}
	id := args.GetAttrVal("id")
	if !id.IsNull() {
		ids := id.AsValueSlice()
		for _, i := range ids {
			n, _ := i.AsBigFloat().Int64()
			req.FilterID = append(req.FilterID, int(n))
		}
	}
	weaknessID := args.GetAttrVal("weakness_id")
	if !weaknessID.IsNull() {
		ids := weaknessID.AsValueSlice()
		for _, i := range ids {
			n, _ := i.AsBigFloat().Int64()
			req.FilterWeaknessID = append(req.FilterWeaknessID, int(n))
		}
	}
	severity := args.GetAttrVal("severity")
	if !severity.IsNull() {
		severities := severity.AsValueSlice()
		for _, s := range severities {
			req.FilterSeverity = append(req.FilterSeverity, s.AsString())
		}
	}
	hackerPublished := args.GetAttrVal("hacker_published")
	if !hackerPublished.IsNull() {
		req.FilterHackerPublished = client.Bool(hackerPublished.True())
	}
	createdAtGT := args.GetAttrVal("created_at__gt")
	if !createdAtGT.IsNull() {
		t, err := time.Parse(time.RFC3339, createdAtGT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at__gt: %w", err)
		}
		req.FilterCreatedAtGT = &t
	}
	createdAtLT := args.GetAttrVal("created_at__lt")
	if !createdAtLT.IsNull() {
		t, err := time.Parse(time.RFC3339, createdAtLT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at__lt: %w", err)
		}
		req.FilterCreatedAtLT = &t
	}
	submittedAtGT := args.GetAttrVal("submitted_at__gt")
	if !submittedAtGT.IsNull() {
		t, err := time.Parse(time.RFC3339, submittedAtGT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse submitted_at__gt: %w", err)
		}
		req.FilterSubmittedAtGT = &t
	}
	submittedAtLT := args.GetAttrVal("submitted_at__lt")
	if !submittedAtLT.IsNull() {
		t, err := time.Parse(time.RFC3339, submittedAtLT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse submitted_at__lt: %w", err)
		}
		req.FilterSubmittedAtLT = &t
	}
	triagedAtGT := args.GetAttrVal("triaged_at__gt")
	if !triagedAtGT.IsNull() {
		t, err := time.Parse(time.RFC3339, triagedAtGT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse triaged_at__gt: %w", err)
		}
		req.FilterTriagedAtGT = &t
	}
	triagedAtLT := args.GetAttrVal("triaged_at__lt")
	if !triagedAtLT.IsNull() {
		t, err := time.Parse(time.RFC3339, triagedAtLT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse triaged_at__lt: %w", err)
		}
		req.FilterTriagedAtLT = &t
	}
	triagedAtNull := args.GetAttrVal("triaged_at__null")
	if !triagedAtNull.IsNull() {
		req.FilterTriagedAtNull = client.Bool(triagedAtNull.True())
	}
	closedAtGT := args.GetAttrVal("closed_at__gt")
	if !closedAtGT.IsNull() {
		t, err := time.Parse(time.RFC3339, closedAtGT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse closed_at__gt: %w", err)
		}
		req.FilterClosedAtGT = &t
	}
	closedAtLT := args.GetAttrVal("closed_at__lt")
	if !closedAtLT.IsNull() {
		t, err := time.Parse(time.RFC3339, closedAtLT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse closed_at__lt: %w", err)
		}
		req.FilterClosedAtLT = &t
	}
	closedAtNull := args.GetAttrVal("closed_at__null")
	if !closedAtNull.IsNull() {
		req.FilterClosedAtNull = client.Bool(closedAtNull.True())
	}
	disclosedAtGT := args.GetAttrVal("disclosed_at__gt")
	if !disclosedAtGT.IsNull() {
		t, err := time.Parse(time.RFC3339, disclosedAtGT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse disclosed_at__gt: %w", err)
		}
		req.FilterDisclosedAtGT = &t
	}
	disclosedAtLT := args.GetAttrVal("disclosed_at__lt")
	if !disclosedAtLT.IsNull() {
		t, err := time.Parse(time.RFC3339, disclosedAtLT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse disclosed_at__lt: %w", err)
		}
		req.FilterDisclosedAtLT = &t
	}
	disclosedAtNull := args.GetAttrVal("disclosed_at__null")
	if !disclosedAtNull.IsNull() {
		req.FilterDisclosedAtNull = client.Bool(disclosedAtNull.True())
	}
	reporterAgreedOnGoingPublic := args.GetAttrVal("reporter_agreed_on_going_public")
	if !reporterAgreedOnGoingPublic.IsNull() {
		req.FilterReporterAgreedOnGoingPublic = client.Bool(reporterAgreedOnGoingPublic.True())
	}
	bountyAwardedAtGT := args.GetAttrVal("bounty_awarded_at__gt")
	if !bountyAwardedAtGT.IsNull() {
		t, err := time.Parse(time.RFC3339, bountyAwardedAtGT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse bounty_awarded_at__gt: %w", err)
		}
		req.FilterBountyAwardedAtGT = &t
	}
	bountyAwardedAtLT := args.GetAttrVal("bounty_awarded_at__lt")
	if !bountyAwardedAtLT.IsNull() {
		t, err := time.Parse(time.RFC3339, bountyAwardedAtLT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse bounty_awarded_at__lt: %w", err)
		}
		req.FilterBountyAwardedAtLT = &t
	}
	bountyAwardedAtNull := args.GetAttrVal("bounty_awarded_at__null")
	if !bountyAwardedAtNull.IsNull() {
		req.FilterBountyAwardedAtNull = client.Bool(bountyAwardedAtNull.True())
	}
	swagAwardedAtGT := args.GetAttrVal("swag_awarded_at__gt")
	if !swagAwardedAtGT.IsNull() {
		t, err := time.Parse(time.RFC3339, swagAwardedAtGT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse swag_awarded_at__gt: %w", err)
		}
		req.FilterSwagAwardedAtGT = &t
	}
	swagAwardedAtLT := args.GetAttrVal("swag_awarded_at__lt")
	if !swagAwardedAtLT.IsNull() {
		t, err := time.Parse(time.RFC3339, swagAwardedAtLT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse swag_awarded_at__lt: %w", err)
		}
		req.FilterSwagAwardedAtLT = &t
	}
	swagAwardedAtNull := args.GetAttrVal("swag_awarded_at__null")
	if !swagAwardedAtNull.IsNull() {
		req.FilterSwagAwardedAtNull = client.Bool(swagAwardedAtNull.True())
	}
	lastReportActivityAtGT := args.GetAttrVal("last_report_activity_at__gt")
	if !lastReportActivityAtGT.IsNull() {
		t, err := time.Parse(time.RFC3339, lastReportActivityAtGT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse last_report_activity_at__gt: %w", err)
		}
		req.FilterLastReportActivityAtGT = &t
	}
	lastReportActivityAtLT := args.GetAttrVal("last_report_activity_at__lt")
	if !lastReportActivityAtLT.IsNull() {
		t, err := time.Parse(time.RFC3339, lastReportActivityAtLT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse last_report_activity_at__lt: %w", err)
		}
		req.FilterLastReportActivityAtLT = &t
	}
	firstProgramActivityAtGT := args.GetAttrVal("first_program_activity_at__gt")
	if !firstProgramActivityAtGT.IsNull() {
		t, err := time.Parse(time.RFC3339, firstProgramActivityAtGT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse first_program_activity_at__gt: %w", err)
		}
		req.FilterFirstProgramActivityAtGT = &t
	}
	firstProgramActivityAtLT := args.GetAttrVal("first_program_activity_at__lt")
	if !firstProgramActivityAtLT.IsNull() {
		t, err := time.Parse(time.RFC3339, firstProgramActivityAtLT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse first_program_activity_at__lt: %w", err)
		}
		req.FilterFirstProgramActivityAtLT = &t
	}
	firstProgramActivityAtNull := args.GetAttrVal("first_program_activity_at__null")
	if !firstProgramActivityAtNull.IsNull() {
		req.FilterFirstProgramActivityAtNull = client.Bool(firstProgramActivityAtNull.True())
	}
	lastProgramActivityAtGT := args.GetAttrVal("last_program_activity_at__gt")
	if !lastProgramActivityAtGT.IsNull() {
		t, err := time.Parse(time.RFC3339, lastProgramActivityAtGT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse last_program_activity_at__gt: %w", err)
		}
		req.FilterLastProgramActivityAtGT = &t
	}
	lastProgramActivityAtLT := args.GetAttrVal("last_program_activity_at__lt")
	if !lastProgramActivityAtLT.IsNull() {
		t, err := time.Parse(time.RFC3339, lastProgramActivityAtLT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse last_program_activity_at__lt: %w", err)
		}
		req.FilterLastProgramActivityAtLT = &t
	}
	lastProgramActivityAtNull := args.GetAttrVal("last_program_activity_at__null")
	if !lastProgramActivityAtNull.IsNull() {
		req.FilterLastProgramActivityAtNull = client.Bool(lastProgramActivityAtNull.True())
	}
	lastActivityAtGT := args.GetAttrVal("last_activity_at__gt")
	if !lastActivityAtGT.IsNull() {
		t, err := time.Parse(time.RFC3339, lastActivityAtGT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse last_activity_at__gt: %w", err)
		}
		req.FilterLastActivityAtGT = &t
	}
	lastActivityAtLT := args.GetAttrVal("last_activity_at__lt")
	if !lastActivityAtLT.IsNull() {
		t, err := time.Parse(time.RFC3339, lastActivityAtLT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse last_activity_at__lt: %w", err)
		}
		req.FilterLastActivityAtLT = &t
	}
	lastPublicActivityAtGT := args.GetAttrVal("last_public_activity_at__gt")
	if !lastPublicActivityAtGT.IsNull() {
		t, err := time.Parse(time.RFC3339, lastPublicActivityAtGT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse last_public_activity_at__gt: %w", err)
		}
		req.FilterLastPublicActivityAtGT = &t
	}
	lastPublicActivityAtLT := args.GetAttrVal("last_public_activity_at__lt")
	if !lastPublicActivityAtLT.IsNull() {
		t, err := time.Parse(time.RFC3339, lastPublicActivityAtLT.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse last_public_activity_at__lt: %w", err)
		}
		req.FilterLastPublicActivityAtLT = &t
	}
	keyword := args.GetAttrVal("keyword")
	if !keyword.IsNull() {
		req.FilterKeyword = client.String(keyword.AsString())
	}
	customFields := args.GetAttrVal("custom_fields")
	if !customFields.IsNull() {
		fields := customFields.AsValueMap()
		req.FilterCustomFields = make(map[string]string, len(fields))
		for k, v := range fields {
			req.FilterCustomFields[k] = v.AsString()
		}
	}
	return &req, nil
}
