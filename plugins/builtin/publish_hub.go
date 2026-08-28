// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package builtin

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	pluginapiv1 "github.com/blackstork-io/blackstork-cli/plugin/pluginapi/v1"
	"github.com/blackstork-io/blackstork-cli/plugins/builtin/hubapi"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeHubPublisher(
	version string,
	loader hubClientLoadFn,
	log *slog.Logger,
	tracer trace.Tracer,
) *plugin.Publisher {
	if tracer == nil {
		tracer = nooptrace.Tracer{}
	}
	return &plugin.Publisher{
		Doc:  "Publish documents to BlackStork cloud",
		Tags: []string{},
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Doc:         "API token",
					Name:        "api_token",
					Secret:      true,
					Type:        cty.String,
					Constraints: constraint.RequiredMeaningful,
				},
				{
					Doc:         "Base URL",
					Name:        "base_url",
					Type:        cty.String,
					Constraints: constraint.RequiredMeaningful,
					DefaultVal:  cty.StringVal("https://run.blackstork.io"),
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{},
		},
		PublishFunc: publishHub(version, loader, log, tracer),
	}
}

func publishHub(version string, loader hubClientLoadFn, log *slog.Logger, tracer trace.Tracer) plugin.PublishFunc {
	return func(ctx context.Context, params *plugin.PublishParams) diagnostics.Diag {
		cli, err := parseHubConfig(params.Config, version, loader)
		if err != nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse config",
				Detail:   err.Error(),
			}}
		}

		encodedDoc := pluginapiv1.EncodeDocument(params.DocumentName, params.Content, params.DataContext)

		publishedDoc, err := cli.UploadDocument(ctx, encodedDoc)
		if err != nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to publish document",
				Detail:   err.Error(),
			}}
		}

		log.InfoContext(ctx, "Published document to hub", "document_id", publishedDoc.ID)
		return nil
	}
}

type hubClientLoadFn func(url, apiToken, version string) hubapi.Client

var defaultHubClientLoader hubClientLoadFn = hubapi.NewClient

func parseHubConfig(cfg *dataspec.Block, version string, loader hubClientLoadFn) (hubapi.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	apiToken := cfg.GetAttrVal("api_token").AsString()
	url := cfg.GetAttrVal("base_url").AsString()

	return loader(url, apiToken, version), nil
}
