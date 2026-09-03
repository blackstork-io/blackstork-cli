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
	"io"
	"log/slog"
	"os"

	"github.com/hashicorp/hcl/v2"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

func makeStdoutPublisher(log *slog.Logger, tracer trace.Tracer) *plugin.Publisher {
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if tracer == nil {
		tracer = nooptrace.Tracer{}
	}
	return &plugin.Publisher{
		Doc:  "Writes the rendered document to standard output.",
		Tags: []string{},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{},
		},
		PublishFunc: publishToStdout(log, tracer),
	}
}

func publishToStdout(log *slog.Logger, tracer trace.Tracer) plugin.PublishFunc {
	return func(ctx context.Context, params *plugin.PublishParams) diagnostics.Diag {
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

		log.InfoContext(ctx, "Publishing content", "format", format)

		bytesCount, err := os.Stdout.Write(content)
		if err != nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to write to stdout",
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
