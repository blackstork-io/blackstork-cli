package builtin

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/hashicorp/hcl/v2"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
)

func makeStdoutPublisher(log *slog.Logger, tracer trace.Tracer) *plugin.Publisher {
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if tracer == nil {
		tracer = nooptrace.Tracer{}
	}
	return &plugin.Publisher{
		Doc:  "Publishes content to stdout",
		Tags: []string{},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{},
		},
		Formats:     []string{"md", "html"},
		PublishFunc: publishToStdout(log, tracer),
	}
}

func publishToStdout(log *slog.Logger, tracer trace.Tracer) plugin.PublishFunc {
	return func(ctx context.Context, params *plugin.PublishParams) diagnostics.Diag {
		document, _ := parseScope(params.DataContext)
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
				Summary:  "No formatted content received",
				Detail:   "formatted content is required",
			}}
		}

		// datactx := params.DataContext
		// datactx["formatted_content"] = plugindata.String(params.FormattedContent.Content)

		content := params.FormattedContent.Content
		format := params.FormattedContent.Format

		log.InfoContext(ctx, "PUBLISHING TO STDOUT", "format", format)

		bytesCount, err := os.Stdout.Write(content)

		if err != nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to write to a stdout",
				Detail:   err.Error(),
			}}
		}
		log.DebugContext(
			ctx, "Content written to stdout",
			"bytes_count", bytesCount,
			"format", format,
		)
		return nil
	}
}
