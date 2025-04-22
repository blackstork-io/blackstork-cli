package builtin

import (
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/blackstork-io/fabric/plugin"
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
			"sleep": makeSleepDataSource(logger),
		},
		ContentProviders: plugin.ContentProviders{
			"toc":         makeTOCContentProvider(),
			"text":        makeTextContentProvider(),
			"title":       makeTitleContentProvider(),
			"code":        makeCodeContentProvider(),
			"blockquote":  makeBlockQuoteContentProvider(),
			"image":       makeImageContentProvider(),
			"list":        makeListContentProvider(),
			"table":       makeTableContentProvider(),
			"sleep":       makeSleepContentProvider(logger),
		},
		Publishers: plugin.Publishers{
			"stdout":     makeStdoutPublisher(logger, tracer),
			"local_file": makeLocalFilePublisher(logger, tracer),
			"hub":        makeHubPublisher(version, defaultHubClientLoader, logger, tracer),
		},
		Formatters: plugin.Formatters{
			"md":   makeMarkdownFormatter(logger, tracer),
			"html": makeHTMLFormatter(logger, tracer),
		},
	}
}
