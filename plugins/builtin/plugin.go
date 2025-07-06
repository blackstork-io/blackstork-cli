package builtin

import (
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/blackstork-io/fabric/internal/plugin"
)

const Name = "blackstork/builtin"

func Plugin(version string, logger *slog.Logger, tracer trace.Tracer) *plugin.Schema {
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
			"sleep": makeSleepDataSource(logger.With("data_source", "sleep")),
		},
		ContentProviders: plugin.ContentProviders{
			"toc":        makeTOCContentProvider(logger.With("content_provider", "toc")),
			"text":       makeTextContentProvider(),
			"title":      makeTitleContentProvider(),
			"code":       makeCodeContentProvider(),
			"blockquote": makeBlockQuoteContentProvider(),
			"image":      makeImageContentProvider(),
			"list":       makeListContentProvider(),
			"table":      makeTableContentProvider(),
			"sleep":      makeSleepContentProvider(logger.With("content_provider", "sleep")),
		},
		Publishers: plugin.Publishers{
			"stdout":     makeStdoutPublisher(logger.With("publisher", "stdout"), tracer),
			"local_file": makeLocalFilePublisher(logger.With("publisher", "local_file"), tracer),
			"hub":        makeHubPublisher(version, defaultHubClientLoader, logger.With("publisher", "hub"), tracer),
		},
		Formatters: plugin.Formatters{
			"md":   makeMarkdownFormatter(logger.With("formatter", "md"), tracer),
			"html": makeHTMLFormatter(logger.With("formatter", "html"), tracer),
		},
	}
}
