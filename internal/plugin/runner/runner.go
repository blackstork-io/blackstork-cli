package runner

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hashicorp/hcl/v2"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin"
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
	tracer trace.Tracer,
) (reg *RunnersRegistry, err error) {
	ctx, span := tracer.Start(ctx, "runner.Load")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "success")
		}
		span.End()
	}()
	//loader := makeLoader(binaryMap, builtin, log, tracer)
	// if diags = loader.loadAll(ctx); diags.HasErrors() {
	// 	return nil, diags
	// }
	// return &RunnersRegistry{
	// 	pluginMap:    loader.pluginMap,
	// 	dataMap:      loader.dataMap,
	// 	contentMap:   loader.contentMap,
	// 	publisherMap: loader.publisherMap,
	// 	formatterMap: loader.formatterMap,
	// }, nil

	return NewRunnersRegistry(ctx, tracer, binaryMap, builtin), nil
}


func NewRunnersRegistry(ctx context.Context,
	tracer trace.Tracer,
	binaryMap map[string]string,
	builtin *plugin.Schema,
) (reg *RunnersRegistry, err error) {

	log := fabctx.Log(ctx)

	ctx, span := tracer.Start(ctx, "runner.NewRunnersRegistry")

	log.DebugContext(ctx, "Loading plugins", "plugins_count", len(binaryMap))

	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "success")
		}
		span.End()
	}()

	reg = &RunnersRegistry{
		pluginMap:    make(map[string]loadedPlugin),
		dataMap:      make(map[string]loadedDataSource),
		contentMap:   make(map[string]loadedContentProvider),
		publisherMap: make(map[string]loadedPublisher),
		formatterMap: make(map[string]loadedFormatter),
	}

	diags = reg.registerPlugin(ctx, l.builtin, nopCloser)
	if diags.HasErrors() {
		diags = append(diags, l.closeAll()...)
		return diags
	}
	for name, binaryPath := range l.binaryMap {
		if diags := l.loadBinary(ctx, name, binaryPath); diags.HasErrors() {
			diags = append(diags, l.closeAll()...)
			return diags
		}
	}
	return nil
}

func (r *RunnersRegistry) registerPlugin(ctx context.Context, tracer trace.Tracer, schema *plugin.Schema, closefn func() error) error {

	log := fabctx.Log(ctx)

	log.DebugContext(
		ctx, "Registering a plugin",
		"name", schema.Name,
		"version", schema.Version,
	)
	if diags := schema.Validate(); diags.HasErrors() {
		log.ErrorContext(
			ctx, "Plugin failed schema validation",
			"name", schema.Name,
			"version", schema.Version,
		)
		return diags
	}
	schema = plugin.WithLogging(schema, log)
	schema = plugin.WithTracing(schema, tracer)
	if found, has := l.pluginMap[schema.Name]; has {
		diags := diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Duplicate plugin name: `%s`", schema.Name),
			Detail: fmt.Sprintf(
				"Plugins `%s@%s` and `%s@%s` have the same schema name",
				schema.Name,
				schema.Version,
				found.Name,
				found.Version,
			),
		}}
		err := found.closefn()
		if err != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Failed to close plugin `%s@%s`", found.Name, found.Version),
				Detail:   err.Error(),
			})
		}
		return diags
	}
	plugin := loadedPlugin{
		closefn: closefn,
		Schema:  schema,
	}
	l.pluginMap[schema.Name] = plugin
	for name, source := range schema.DataSources {
		if diags := l.registerDataSource(ctx, name, schema, source); diags.HasErrors() {
			return diags
		}
	}
	for name, provider := range schema.ContentProviders {
		if diags := l.registerContentProvider(ctx, name, schema, provider); diags.HasErrors() {
			return diags
		}
	}
	for name, publisher := range schema.Publishers {
		if diags := l.registerPublisher(ctx, name, schema, publisher); diags.HasErrors() {
			return diags
		}
	}
	for name, formatter := range schema.Formatters {
		if diags := l.registerFormatter(ctx, name, schema, formatter); diags.HasErrors() {
			return diags
		}
	}
	return nil
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
