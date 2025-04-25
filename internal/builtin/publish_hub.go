package builtin

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"

	"github.com/blackstork-io/fabric/internal/builtin/hubapi"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/constraint"
	pluginapiv1 "github.com/blackstork-io/fabric/plugin/pluginapi/v1"
)

func makeHubPublisher(
	version string,
	loader hubClientLoadFn,
	logger *slog.Logger,
	tracer trace.Tracer,
) *plugin.Publisher {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if tracer == nil {
		tracer = nooptrace.Tracer{}
	}
	return &plugin.Publisher{
		Doc:  "Publish documents to Blackstork Hub.",
		Tags: []string{},
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Doc:         "API url.",
					Name:        "api_url",
					Type:        cty.String,
					Constraints: constraint.RequiredMeaningful,
				},
				{
					Doc:         "API url.",
					Name:        "api_token",
					Type:        cty.String,
					Constraints: constraint.RequiredMeaningful,
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "title",
					Doc:         "Hub Document title override. By default uses title configured in the document.",
					Type:        cty.String,
					Constraints: constraint.Meaningful,
				},
			},
		},
		Formats:     []string{},
		PublishFunc: publishHub(version, loader, logger, tracer),
	}
}

func publishHub(version string, loader hubClientLoadFn, logger *slog.Logger, tracer trace.Tracer) plugin.PublishFunc {
	return func(ctx context.Context, params *plugin.PublishParams) diagnostics.Diag {
		cli, err := parseHubConfig(params.Config, version, loader)
		if err != nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse config",
				Detail:   err.Error(),
			}}
		}

		content, _ := parseScope(params.DataContext)
		if content == nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse document",
				Detail:   "document is required",
			}}
		}

		title := "Untitled"
		if found, ok := firstTitle(content); ok {
			title = found
		}
		if attr := params.Args.GetAttrVal("title"); !attr.IsNull() {
			title, err = renderText(attr.AsString(), params.DataContext)
			if err != nil {
				return diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Failed to template publish title",
					Detail:   err.Error(),
				}}
			}
		}

		doc, err := cli.CreateDocument(ctx, &hubapi.DocumentParams{
			Title: title,
		})
		if err != nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to publish document",
				Detail:   err.Error(),
			}}
		}

		uploadedContent, err := cli.UploadDocumentContent(ctx, doc.ID, pluginapiv1.EncodeContent(content))
		if err != nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to publish document",
				Detail:   err.Error(),
			}}
		}

		logger.Info("Published to Hub",
			slog.Group("document",
				slog.String("id", doc.ID),
				slog.String("title", doc.Title),
				slog.Group("content",
					slog.String("id", uploadedContent.ID),
				),
				slog.Time("created_at", doc.CreatedAt),
			),
		)
		return nil
	}
}

type hubClientLoadFn func(apiURL, apiToken, version string) hubapi.Client

var defaultHubClientLoader hubClientLoadFn = hubapi.NewClient

func parseHubConfig(cfg *dataspec.Block, version string, loader hubClientLoadFn) (hubapi.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	apiURL := cfg.GetAttrVal("api_url").AsString()
	apiToken := cfg.GetAttrVal("api_token").AsString()

	return loader(apiURL, apiToken, version), nil
}
