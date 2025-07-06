package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/hashicorp/hcl/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin"
	pluginapiv1 "github.com/blackstork-io/fabric/internal/plugin/pluginapi/v1"
)

type loadedPlugin struct {
	closefn func() error
	*plugin.Schema
}

type loadedDataSource struct {
	plugin *plugin.Schema
	*plugin.DataSource
}

type loadedContentProvider struct {
	plugin *plugin.Schema
	*plugin.ContentProvider
}

type loadedPublisher struct {
	plugin *plugin.Schema
	*plugin.Publisher
}

type loadedFormatter struct {
	plugin *plugin.Schema
	*plugin.Formatter
}

type loader struct {
	logger       *slog.Logger
	tracer       trace.Tracer
	binaryMap    map[string]string
	builtin      *plugin.Schema
	pluginMap    map[string]loadedPlugin
	dataMap      map[string]loadedDataSource
	contentMap   map[string]loadedContentProvider
	publisherMap map[string]loadedPublisher
	formatterMap map[string]loadedFormatter
}

func makeLoader(
	binaryMap map[string]string,
	builtin *plugin.Schema,
	logger *slog.Logger,
	tracer trace.Tracer,
) *loader {
	return &loader{
		tracer:       tracer,
		logger:       logger,
		binaryMap:    binaryMap,
		builtin:      builtin,
		pluginMap:    make(map[string]loadedPlugin),
		dataMap:      make(map[string]loadedDataSource),
		contentMap:   make(map[string]loadedContentProvider),
		publisherMap: make(map[string]loadedPublisher),
		formatterMap: make(map[string]loadedFormatter),
	}
}

func nopCloser() error {
	return nil
}

func (l *loader) loadAll(ctx context.Context) (diags diagnostics.Diag) {
	ctx, span := l.tracer.Start(ctx, "loader.loadAll")
	l.logger.DebugContext(ctx, "Loading all plugins", "plugins_count", len(l.binaryMap))
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		} else {
			span.SetStatus(codes.Ok, "success")
		}
		span.End()
	}()

	diags = l.registerPlugin(ctx, l.builtin, nopCloser)
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

func (l *loader) closeAll() diagnostics.Diag {
	var diags diagnostics.Diag
	for _, p := range l.pluginMap {
		if err := p.closefn(); err != nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Failed to close plugin",
				Detail:   fmt.Sprintf("Failed to close plugin %s@%s: %v", p.Name, p.Version, err),
			})
		}
	}
	return diags
}

func (l *loader) registerDataSource(
	ctx context.Context,
	name string,
	schema *plugin.Schema,
	ds *plugin.DataSource,
) diagnostics.Diag {
	l.logger.DebugContext(
		ctx, "Registering a data source",
		"name", name,
		"plugin", schema.Name,
		"version", schema.Version,
	)
	if found, has := l.dataMap[name]; has {
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Duplicate data source",
			Detail: fmt.Sprintf(
				"Data source `%s` is provided by two plugins: `%s@%s` and `%s@%s`",
				name,
				schema.Name,
				schema.Version,
				found.plugin.Name,
				found.plugin.Version,
			),
		}}
	}
	l.dataMap[name] = loadedDataSource{
		plugin:     schema,
		DataSource: ds,
	}
	return nil
}

func (l *loader) registerContentProvider(
	ctx context.Context,
	name string,
	schema *plugin.Schema,
	cp *plugin.ContentProvider,
) diagnostics.Diag {
	l.logger.DebugContext(
		ctx,
		"Registering a content provider",
		"name", name,
		"plugin", schema.Name,
		"version", schema.Version,
	)
	if found, has := l.contentMap[name]; has {
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Duplicate content provider",
			Detail: fmt.Sprintf(
				"Content provider `%s` provided by two plugins: `%s@%s` and `%s@%s`",
				name,
				schema.Name,
				schema.Version,
				found.plugin.Name,
				found.plugin.Version,
			),
		}}
	}
	l.contentMap[name] = loadedContentProvider{
		plugin: schema,
		ContentProvider: &plugin.ContentProvider{
			Config: cp.Config,
			Args:   cp.Args,
			Doc:    cp.Doc,
			Tags:   cp.Tags,
			ContentFunc: func(ctx context.Context, params *plugin.ProvideContentParams) (*plugin.ContentProviderResult, diagnostics.Diag) {
				return schema.ProvideContent(ctx, name, params)
			},
		},
	}
	return nil
}

func (l *loader) registerPublisher(
	ctx context.Context,
	name string,
	schema *plugin.Schema,
	pub *plugin.Publisher,
) diagnostics.Diag {
	l.logger.DebugContext(
		ctx,
		"Registering a publisher",
		"name",
		name,
		"plugin",
		schema.Name,
		"version",
		schema.Version,
	)
	if found, has := l.publisherMap[name]; has {
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Duplicate publisher",
			Detail: fmt.Sprintf(
				"Publisher `%s` provided by two plugins: `%s@%s` and `%s@%s`",
				name,
				schema.Name,
				schema.Version,
				found.plugin.Name,
				found.plugin.Version,
			),
		}}
	}
	l.publisherMap[name] = loadedPublisher{
		plugin:    schema,
		Publisher: pub,
	}
	return nil
}

func (l *loader) registerFormatter(
	ctx context.Context,
	name string,
	schema *plugin.Schema,
	frmt *plugin.Formatter,
) diagnostics.Diag {
	l.logger.DebugContext(
		ctx,
		"Registering a formatter",
		"name", name,
		"plugin", schema.Name,
		"version", schema.Version,
	)
	if found, has := l.publisherMap[name]; has {
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Duplicate formatter",
			Detail: fmt.Sprintf(
				"Formatter '%s' provided by plugins %s@%s and %s@%s at the same time",
				name,
				schema.Name,
				schema.Version,
				found.plugin.Name,
				found.plugin.Version,
			),
		}}
	}
	l.formatterMap[name] = loadedFormatter{
		plugin:    schema,
		Formatter: frmt,
	}
	return nil
}

func (l *loader) registerPlugin(ctx context.Context, schema *plugin.Schema, closefn func() error) diagnostics.Diag {
	l.logger.DebugContext(
		ctx, "Registering a plugin",
		"name", schema.Name,
		"version", schema.Version,
	)
	if diags := schema.Validate(); diags.HasErrors() {
		l.logger.ErrorContext(
			ctx, "Schema validation errors while registering a plugin",
			"name", schema.Name,
			"version", schema.Version,
		)
		return diags
	}
	schema = plugin.WithLogging(schema, l.logger)
	schema = plugin.WithTracing(schema, l.tracer)
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

func (l *loader) loadBinary(ctx context.Context, name, binaryPath string) (diags diagnostics.Diag) {
	ctx, span := l.tracer.Start(ctx, "loader.loadBinary", trace.WithAttributes(
		attribute.String("name", name),
	))
	l.logger.InfoContext(ctx, "Loading a plugin", "name", name, "path", binaryPath)
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		} else {
			span.SetStatus(codes.Ok, "success")
		}
		span.End()
	}()
	if info, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("The binary for plugin `%s` not found", name),
			Detail:   fmt.Sprintf("Executable not found at `%s`", binaryPath),
		}}
	} else if info.IsDir() {
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Binary path for plugin `%s` is a directory", name),
			Detail:   fmt.Sprintf("Path `%s` is a directory", binaryPath),
		}}
	}
	p, close, err := pluginapiv1.NewClient(name, binaryPath, l.logger)
	if err != nil {
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Failed to load plugin `%s`", name),
			Detail:   err.Error(),
		}}
	}
	return l.registerPlugin(ctx, p, close)
}
