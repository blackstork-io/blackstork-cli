package plugin

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

// WithLogging wraps the plugin with logging instrumentation.
func WithLogging(plugin *Schema, log *slog.Logger) *Schema {
	log = log.With("plugin", plugin.Name)
	plugin.ContentProviders = makeContentProvidersLogging(plugin.ContentProviders, log)
	plugin.DataSources = makeDataSourcesLogging(plugin.DataSources, log)
	plugin.Formatters = makeFormattersLogging(plugin.Formatters, log)
	plugin.Publishers = makePublishersLogging(plugin.Publishers, log)
	return plugin
}

func makeContentProvidersLogging(providers ContentProviders, log *slog.Logger) ContentProviders {
	result := make(ContentProviders)
	for name, provider := range providers {
		provider.ContentFunc = makeContentProviderLogging(name, *provider, log)
		result[name] = provider
	}
	return result
}

func makeDataSourcesLogging(sources DataSources, log *slog.Logger) DataSources {
	result := make(DataSources)
	for name, source := range sources {
		source.DataFunc = makeDataSourceLogging(name, *source, log)
		result[name] = source
	}
	return result
}

func makePublishersLogging(publishers Publishers, log *slog.Logger) Publishers {
	result := make(Publishers)
	for name, publisher := range publishers {
		publisher.PublishFunc = makePublisherLogging(name, *publisher, log)
		result[name] = publisher
	}
	return result
}

func makePublisherLogging(name string, publisher Publisher, log *slog.Logger) PublishFunc {
	next := publisher.PublishFunc
	return func(ctx context.Context, params *PublishParams) diagnostics.Diag {
		log.DebugContext(
			ctx, "Executing a publisher",
			slog.String("publisher", name),
			slog.Any("supported_formats", publisher.Formats),
			slog.Any("config", logDataBlockValue(params.Config)),
			slog.Any("args", logDataBlockValue(params.Args)),
			slog.String("document", params.DocumentName),
		)
		return next(ctx, params)
	}
}

func makeFormattersLogging(formatters Formatters, log *slog.Logger) Formatters {
	result := make(Formatters)
	for name, formatter := range formatters {
		formatter.FormatFunc = makeFormatterLogging(*formatter, log)
		result[name] = formatter
	}
	return result
}

func makeFormatterLogging(formatter Formatter, log *slog.Logger) FormatFunc {
	next := formatter.FormatFunc
	return func(ctx context.Context, params *FormatParams) (*FormattedContent, diagnostics.Diag) {
		log.DebugContext(ctx, "Executing a formatter",
			slog.Any("format", formatter.Format),
			slog.Any("file_ext", formatter.FileExt),
			slog.Any("config", logDataBlockValue(params.Config)),
			slog.Any("args", logDataBlockValue(params.Args)),
		)
		return next(ctx, params)
	}
}

func makeContentProviderLogging(name string, provider ContentProvider, log *slog.Logger) ProvideContentFunc {
	next := provider.ContentFunc
	return func(ctx context.Context, params *ProvideContentParams) (*ContentProviderResult, diagnostics.Diag) {
		log.DebugContext(ctx, "Executing a content provider",
			slog.String("provider", name),
			slog.Any("config", logDataBlockValue(params.Config)),
			slog.Any("args", logDataBlockValue(params.Args)),
		)
		return next(ctx, params)
	}
}

func makeDataSourceLogging(name string, source DataSource, log *slog.Logger) RetrieveDataFunc {
	next := source.DataFunc
	return func(ctx context.Context, params *RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		log.DebugContext(ctx, "Executing a data source",
			slog.String("data_source", name),
			slog.Any("config", logDataBlockValue(params.Config)),
			slog.Any("args", logDataBlockValue(params.Args)),
		)
		return next(ctx, params)
	}
}

func logDataBlockValue(value *dataspec.Block) slog.Value {
	if value == nil {
		return slog.Value{}
	}
	attrs := make([]slog.Attr, 0, len(value.Blocks)+len(value.Attrs))
	for _, b := range value.Blocks {
		if b == nil {
			continue
		}
		attrs = append(attrs, slog.Attr{
			Key:   "block_" + strings.Join(b.Header, "_"),
			Value: logDataBlockValue(b),
		})
	}
	for _, a := range value.Attrs {
		if a == nil {
			continue
		}
		attrs = append(attrs, logDataAttr(a))
	}
	return slog.GroupValue(attrs...)
}

func logDataAttr(attr *dataspec.Attr) (val slog.Attr) {
	val.Key = attr.Name
	if attr.Secret {
		val.Value = slog.StringValue("<secret>")
	} else {
		val.Value = logCtyValue(attr.Value)
	}
	return
}

func logCtyValue(value cty.Value) slog.Value {
	if value.IsNull() {
		return slog.Value{}
	}
	switch {
	case value.Type() == cty.String:
		return slog.StringValue(value.AsString())
	case value.Type() == cty.Number:
		f, _ := value.AsBigFloat().Float64()
		return slog.Float64Value(f)
	case value.Type() == cty.Bool:
		return slog.BoolValue(value.True())
	case value.Type().IsListType() || value.Type().IsTupleType() || value.Type().IsSetType():
		return logCtyListValue(value.AsValueSlice())
	case value.Type().IsMapType() || value.Type().IsObjectType():
		return logCtyMapValue(value.AsValueMap())
	default:
		return slog.Value{}
	}
}

func logCtyListValue(values []cty.Value) slog.Value {
	attrs := []slog.Attr{}
	for i, v := range values {
		k := strconv.Itoa(i)
		switch {
		case v.Type() == cty.String:
			attrs = append(attrs, slog.String(k, v.AsString()))
		case v.Type() == cty.Number:
			f, _ := v.AsBigFloat().Float64()
			attrs = append(attrs, slog.Float64(k, f))
		case v.Type() == cty.Bool:
			attrs = append(attrs, slog.Bool(k, v.True()))
		case v.Type().IsListType() || v.Type().IsTupleType() || v.Type().IsSetType():
			attrs = append(attrs, slog.Any(k, logCtyListValue(v.AsValueSlice())))
		case v.Type().IsMapType() || v.Type().IsObjectType():
			attrs = append(attrs, slog.Any(k, logCtyMapValue(v.AsValueMap())))
		default:
			attrs = append(attrs, slog.String(k, "<unknown>"))
		}
	}
	return slog.GroupValue(attrs...)
}

func logCtyMapValue(m map[string]cty.Value) slog.Value {
	attrs := []slog.Attr{}
	for k, v := range m {
		switch {
		case v.Type() == cty.String:
			attrs = append(attrs, slog.String(k, v.AsString()))
		case v.Type() == cty.Number:
			f, _ := v.AsBigFloat().Float64()
			attrs = append(attrs, slog.Float64(k, f))
		case v.Type() == cty.Bool:
			attrs = append(attrs, slog.Bool(k, v.True()))
		case v.Type().IsListType() || v.Type().IsTupleType() || v.Type().IsSetType():
			attrs = append(attrs, slog.Any(k, logCtyListValue(v.AsValueSlice())))
		case v.Type().IsMapType() || v.Type().IsObjectType():
			attrs = append(attrs, slog.Any(k, logCtyMapValue(v.AsValueMap())))
		default:
			attrs = append(attrs, slog.String(k, "<unknown>"))
		}
	}
	return slog.GroupValue(attrs...)
}
