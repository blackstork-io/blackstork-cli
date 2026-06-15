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
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/blackstork-io/blackstork-cli/plugin"
)

const Name = "blackstork/builtin"

func Plugin(version string, log *slog.Logger, tracer trace.Tracer) *plugin.Schema {
	return &plugin.Schema{
		Name:    Name,
		Version: version,
		DataSources: plugin.DataSources{
			"csv":   makeCSVDataSource(),
			"txt":   makeTXTDataSource(),
			"rss":   makeRSSDataSource(),
			"json":  makeJSONDataSource(),
			"yaml":  makeYAMLDataSource(),
			"http":  makeHTTPDataSource(version),
			"sleep": makeSleepDataSource(log.With("data_source", "sleep")),
		},
		ContentProviders: plugin.ContentProviders{
			"toc":        makeTOCContentProvider(log.With("content_provider", "toc")),
			"text":       makeTextContentProvider(),
			"title":      makeTitleContentProvider(),
			"code":       makeCodeContentProvider(),
			"blockquote": makeBlockQuoteContentProvider(),
			"image":      makeImageContentProvider(),
			"list":       makeListContentProvider(),
			"table":      makeTableContentProvider(),
			"sleep":      makeSleepContentProvider(log.With("content_provider", "sleep")),
			"llm_text":   makeLLMTextContentProvider(),
		},
		Publishers: plugin.Publishers{
			"stdout":     makeStdoutPublisher(log.With("publisher", "stdout"), tracer),
			"local_file": makeLocalFilePublisher(log.With("publisher", "local_file"), tracer),
			"hub":        makeHubPublisher(version, defaultHubClientLoader, log.With("publisher", "hub"), tracer),
		},
		Formatters: plugin.Formatters{
			"md":   makeMarkdownFormatter(log.With("formatter", "md"), tracer),
			"html": makeHTMLFormatter(log.With("formatter", "html"), tracer),
		},
	}
}
