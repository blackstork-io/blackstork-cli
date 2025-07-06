package runner

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hashicorp/hcl/v2"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
)

type RunnersRegistry struct {
	pluginMap    map[string]loadedPlugin
	dataMap      map[string]loadedDataSource
	contentMap   map[string]loadedContentProvider
	publisherMap map[string]loadedPublisher
	formatterMap map[string]loadedFormatter
}

func Load(
	ctx context.Context,
	binaryMap map[string]string,
	builtin *plugin.Schema,
	log *slog.Logger,
	tracer trace.Tracer,
) (_ *RunnersRegistry, diags diagnostics.Diag) {
	ctx, span := tracer.Start(ctx, "runner.Load")
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	loader := makeLoader(binaryMap, builtin, log, tracer)
	if diags = loader.loadAll(ctx); diags.HasErrors() {
		return nil, diags
	}
	return &RunnersRegistry{
		pluginMap:    loader.pluginMap,
		dataMap:      loader.dataMap,
		contentMap:   loader.contentMap,
		publisherMap: loader.publisherMap,
		formatterMap: loader.formatterMap,
	}, nil
}

func (m *RunnersRegistry) Plugins() []*plugin.Schema {
	var plugins []*plugin.Schema
	for _, p := range m.pluginMap {
		plugins = append(plugins, p.Schema)
	}
	return plugins
}

func (m *RunnersRegistry) Schema(name string) (*plugin.Schema, bool) {
	p, ok := m.pluginMap[name]
	if !ok {
		return nil, false
	}
	return p.Schema, true
}

func (m *RunnersRegistry) DataSource(name string) (*plugin.DataSource, bool) {
	source, ok := m.dataMap[name]
	if !ok {
		return nil, false
	}
	return source.DataSource, true
}

func (m *RunnersRegistry) ContentProvider(name string) (*plugin.ContentProvider, bool) {
	provider, ok := m.contentMap[name]
	if !ok {
		return nil, false
	}
	return provider.ContentProvider, true
}

func (m *RunnersRegistry) Publisher(name string) (*plugin.Publisher, bool) {
	publisher, ok := m.publisherMap[name]
	if !ok {
		return nil, false
	}
	return publisher.Publisher, true
}

func (m *RunnersRegistry) Formatter(name string) (*plugin.Formatter, bool) {
	formatter, ok := m.formatterMap[name]
	if !ok {
		return nil, false
	}
	return formatter.Formatter, true
}

func (m *RunnersRegistry) Close() diagnostics.Diag {
	var diags diagnostics.Diag
	for _, p := range m.pluginMap {
		if err := p.closefn(); err != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  fmt.Sprintf("Failed to close plugin '%s'", p.Name),
				Detail:   err.Error(),
			})
		}
	}
	return diags
}
