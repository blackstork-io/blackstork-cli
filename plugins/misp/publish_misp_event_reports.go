// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package misp

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugins/misp/client"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeMispEventReportsPublisher(loader ClientLoaderFn) *plugin.Publisher {
	return &plugin.Publisher{
		Doc:    "Publishes content to misp event reports",
		Tags:   []string{},
		Config: makeDataSourceConfig(),
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "event_id",
					Type:        cty.String,
					Constraints: constraint.RequiredMeaningful,
				},
				{
					Name:        "name",
					Type:        cty.String,
					Constraints: constraint.RequiredMeaningful,
				},
				{
					Name: "distribution",
					Type: cty.String,
					OneOf: []cty.Value{
						cty.StringVal("0"),
						cty.StringVal("1"),
						cty.StringVal("2"),
						cty.StringVal("3"),
						cty.StringVal("4"),
						cty.StringVal("5"),
					},
				},
				{
					Name: "sharing_group_id",
					Type: cty.String,
				},
			},
		},
		Formats:     []string{"md"},
		PublishFunc: publishEventReport(loader),
	}
}

func parseContent(data plugindata.Map) (document *plugin.ContentSection) {
	documentMap, ok := data["document"]
	if !ok {
		return document
	}
	contentMap, ok := documentMap.(plugindata.Map)["content"]
	if !ok {
		return document
	}
	content, err := plugin.ParseContentData(contentMap.(plugindata.Map))
	if err != nil {
		return document
	}
	document = content.(*plugin.ContentSection)
	return document
}

func publishEventReport(loader ClientLoaderFn) plugin.PublishFunc {
	// logger := slog.Default()
	// tracer := nooptrace.Tracer{}
	return func(ctx context.Context, params *plugin.PublishParams) diagnostics.Diag {
		document := parseContent(params.DataContext)
		if document == nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse document",
				Detail:   "document is required",
			}}
		}

		if params.FormattedContent == nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "No content provided",
				Detail:   "content must be provided for MISP publisher",
			}}
		}

		//datactx := params.DataContext
		//datactx["format"] = plugindata.String(format)
		// var printer print.Printer
		// switch format {
		// case "md":
		// 	printer = mdprint.New()
		// default:
		// 	return diagnostics.Diag{{
		// 		Severity: hcl.DiagError,
		// 		Summary:  "Unsupported format",
		// 		Detail:   "Only md format is supported",
		// 	}}
		// }
		// printer = print.WithLogging(printer, logger, slog.String("format", format))
		// printer = print.WithTracing(printer, tracer, attribute.String("format", format))
		//
		buff := bytes.NewBuffer(nil)
		// err := printer.Print(ctx, buff, document)
		// if err != nil {
		// 	return diagnostics.Diag{{
		// 		Severity: hcl.DiagError,
		// 		Summary:  "Failed to render the report",
		// 		Detail:   err.Error(),
		// 	}}
		// }

		cli := loader(params.Config)

		timestamp := fmt.Sprintf("%d", time.Now().Unix())
		report := client.AddEventReportRequest{
			Uuid:      uuid.New().String(),
			EventId:   params.Args.GetAttrVal("event_id").AsString(),
			Name:      params.Args.GetAttrVal("name").AsString(),
			Content:   buff.String(),
			Timestamp: &timestamp,
			Deleted:   false,
		}
		distribution := params.Args.GetAttrVal("distribution")
		if !distribution.IsNull() {
			distributionStr := distribution.AsString()
			report.Distribution = &distributionStr
		}
		sharingGroupId := params.Args.GetAttrVal("sharing_group_id")
		if !sharingGroupId.IsNull() {
			sharingGroupIdStr := sharingGroupId.AsString()
			report.SharingGroupId = &sharingGroupIdStr
		}

		slog.InfoContext(ctx, "Publish to misp event reports", "filename", report.Name)

		resp, err := cli.AddEventReport(ctx, report)
		if err != nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to add event report",
				Detail:   err.Error(),
			}}
		}

		slog.InfoContext(
			ctx,
			"Successfully added report",
			"id",
			resp.EventReport.Id,
			"uuid",
			resp.EventReport.Uuid,
			"event_id",
			resp.EventReport.EventId,
			"name",
			resp.EventReport.Name,
		)
		return nil
	}
}
