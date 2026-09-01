// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package eval

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/plugin"
	pluginapiv1 "github.com/blackstork-io/blackstork-cli/plugin/pluginapi/v1"
)

type DataSources interface {
	DataSource(name string) (*plugin.DataSource, bool)
	DataSources() iter.Seq2[string, *plugin.DataSource]
	DataSourceSchema(name string) (*plugin.Schema, bool)
}

type ContentProviders interface {
	ContentProvider(name string) (*plugin.ContentProvider, bool)
	ContentProviders() iter.Seq2[string, *plugin.ContentProvider]
}

type Publishers interface {
	Publisher(name string) (*plugin.Publisher, bool)
	Publishers() iter.Seq2[string, *plugin.Publisher]
}

type Formatters interface {
	Formatter(name string) (*plugin.Formatter, bool)
	Formatters() iter.Seq2[string, *plugin.Formatter]
}

type Runners interface {
	DataSources
	ContentProviders
	Publishers
	Formatters
}

type loadedPlugin struct {
	closefn func() error
	*plugin.Schema
}

type loadedDataSource struct {
	schema *plugin.Schema
	*plugin.DataSource
}

type loadedContentProvider struct {
	schema *plugin.Schema
	*plugin.ContentProvider
}

type loadedPublisher struct {
	schema *plugin.Schema
	*plugin.Publisher
}

type loadedFormatter struct {
	schema *plugin.Schema
	*plugin.Formatter
}

type RunnersRegistry interface {
	Runners

	Schemas() []*plugin.Schema
	CloseAll() error
}

type runnersRegistry struct {
	pluginMap    map[string]loadedPlugin
	dataMap      map[string]loadedDataSource
	contentMap   map[string]loadedContentProvider
	publisherMap map[string]loadedPublisher
	formatterMap map[string]loadedFormatter
}

func nopCloser() error {
	return nil
}

func LoadRunnersRegistry(
	ctx context.Context,
	binaryMap map[string]string,
	builtin *plugin.Schema,
) (reg RunnersRegistry, err error) {
	tracer := appctx.Tracer(ctx)
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
	// loader := makeLoader(binaryMap, builtin, log, tracer)
	// if diags = loader.loadAll(ctx); diags.HasErrors() {
	// 	return nil, diags
	// }
	// return &runnersRegistry{
	// 	pluginMap:    loader.pluginMap,
	// 	dataMap:      loader.dataMap,
	// 	contentMap:   loader.contentMap,
	// 	publisherMap: loader.publisherMap,
	// 	formatterMap: loader.formatterMap,
	// }, nil

	return newRegistry(ctx, binaryMap, builtin)
}

func newRegistry(
	ctx context.Context,
	binaryMap map[string]string,
	builtin *plugin.Schema,
) (reg *runnersRegistry, err error) {
	log := appctx.Log(ctx)
	tracer := appctx.Tracer(ctx)

	ctx, span := tracer.Start(ctx, "runner.NewrunnersRegistry")

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

	reg = &runnersRegistry{
		pluginMap:    make(map[string]loadedPlugin),
		dataMap:      make(map[string]loadedDataSource),
		contentMap:   make(map[string]loadedContentProvider),
		publisherMap: make(map[string]loadedPublisher),
		formatterMap: make(map[string]loadedFormatter),
	}

	err = reg.registerPlugin(ctx, builtin, nopCloser)
	if err != nil {
		closeErr := reg.closeAll(ctx)
		err = errors.Join(err, closeErr)
		return nil, err
	}
	for name, binaryPath := range binaryMap {
		err = reg.loadBinary(ctx, name, binaryPath)
		if err != nil {
			closeErr := reg.closeAll(ctx)
			err = errors.Join(err, closeErr)
			return nil, err
		}
	}
	return reg, nil
}

func (reg *runnersRegistry) registerPlugin(
	ctx context.Context,
	schema *plugin.Schema,
	closefn func() error,
) error {
	tracer := appctx.Tracer(ctx)
	log := appctx.Log(ctx)

	log.DebugContext(
		ctx, "Registering plugin",
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

	if found, ok := reg.pluginMap[schema.Name]; ok {
		log.ErrorContext(
			ctx, "Duplicate plugin schema name",
			"new_name", schema.Name,
			"new_version", schema.Version,
			"found_name", found.Name,
			"found_version", found.Version,
		)
		err := fmt.Errorf(
			"plugins `%s@%s` and `%s@%s` have the same schema name",
			schema.Name,
			schema.Version,
			found.Name,
			found.Version,
		)
		errClosing := found.closefn()
		if errClosing != nil {
			log.ErrorContext(
				ctx, "Failed to close the plugin",
				"name", found.Name,
				"version", found.Version,
			)
			err = errors.Join(err, errClosing)
		}
		return err
	}
	plugin := loadedPlugin{
		closefn: closefn,
		Schema:  schema,
	}
	reg.pluginMap[schema.Name] = plugin
	for name, source := range schema.DataSources {
		err := reg.registerDataSource(ctx, name, schema, source)
		if err != nil {
			return err
		}
	}
	for name, provider := range schema.ContentProviders {
		err := reg.registerContentProvider(ctx, name, schema, provider)
		if err != nil {
			return err
		}
	}
	for name, publisher := range schema.Publishers {
		err := reg.registerPublisher(ctx, name, schema, publisher)
		if err != nil {
			return err
		}
	}
	for name, formatter := range schema.Formatters {
		err := reg.registerFormatter(ctx, name, schema, formatter)
		if err != nil {
			return err
		}
	}
	return nil
}

func (reg *runnersRegistry) Plugins() []*plugin.Schema {
	var plugins []*plugin.Schema
	for _, p := range reg.pluginMap {
		plugins = append(plugins, p.Schema)
	}
	return plugins
}

func (reg *runnersRegistry) Schema(name string) (*plugin.Schema, bool) {
	p, ok := reg.pluginMap[name]
	if !ok {
		return nil, false
	}
	return p.Schema, true
}

func (reg *runnersRegistry) DataSource(name string) (*plugin.DataSource, bool) {
	source, ok := reg.dataMap[name]
	if !ok {
		return nil, false
	}
	return source.DataSource, true
}

func (reg *runnersRegistry) DataSourceSchema(name string) (*plugin.Schema, bool) {
	source, ok := reg.dataMap[name]
	if !ok {
		return nil, false
	}
	return source.schema, true
}

func (reg *runnersRegistry) DataSources() iter.Seq2[string, *plugin.DataSource] {
	return func(yield func(string, *plugin.DataSource) bool) {
		for k, v := range reg.dataMap {
			val := v.DataSource
			if !yield(k, val) {
				return
			}
		}
	}
}

func (reg *runnersRegistry) ContentProvider(name string) (*plugin.ContentProvider, bool) {
	provider, ok := reg.contentMap[name]
	if !ok {
		return nil, false
	}
	return provider.ContentProvider, true
}

func (reg *runnersRegistry) ContentProviders() iter.Seq2[string, *plugin.ContentProvider] {
	return func(yield func(string, *plugin.ContentProvider) bool) {
		for k, v := range reg.contentMap {
			val := v.ContentProvider
			if !yield(k, val) {
				return
			}
		}
	}
}

func (reg *runnersRegistry) Publisher(name string) (*plugin.Publisher, bool) {
	publisher, ok := reg.publisherMap[name]
	if !ok {
		return nil, false
	}
	return publisher.Publisher, true
}

func (reg *runnersRegistry) Publishers() iter.Seq2[string, *plugin.Publisher] {
	return func(yield func(string, *plugin.Publisher) bool) {
		for k, v := range reg.publisherMap {
			val := v.Publisher
			if !yield(k, val) {
				return
			}
		}
	}
}

func (reg *runnersRegistry) Formatter(name string) (*plugin.Formatter, bool) {
	formatter, ok := reg.formatterMap[name]
	if !ok {
		return nil, false
	}
	return formatter.Formatter, true
}

func (reg *runnersRegistry) Formatters() iter.Seq2[string, *plugin.Formatter] {
	return func(yield func(string, *plugin.Formatter) bool) {
		for k, v := range reg.formatterMap {
			val := v.Formatter
			if !yield(k, val) {
				return
			}
		}
	}
}

func (reg *runnersRegistry) CloseAll() error {
	for _, p := range reg.pluginMap {
		if err := p.closefn(); err != nil {
			return err
		}
	}
	return nil
}

func (reg *runnersRegistry) Schemas() []*plugin.Schema {
	res := []*plugin.Schema{}
	for _, p := range reg.pluginMap {
		res = append(res, p.Schema)
	}
	return res
}

func (reg *runnersRegistry) registerDataSource(
	ctx context.Context,
	name string,
	schema *plugin.Schema,
	ds *plugin.DataSource,
) error {
	log := appctx.Log(ctx)
	log.DebugContext(
		ctx, "Registering data source",
		"name", name,
		"plugin", schema.Name,
		"version", schema.Version,
	)
	if found, ok := reg.dataMap[name]; ok {
		log.ErrorContext(
			ctx, "Data source with the same name found in two plugins",
			"name", name,
			"plugin1", schema.Name,
			"plugin1_version", schema.Version,
			"plugin2", found.schema.Name,
			"plugin2_version", found.schema.Version,
		)
		return fmt.Errorf(
			"data source `%s` is found in two plugins: `%s@%s` and `%s@%s`",
			name,
			schema.Name,
			schema.Version,
			found.schema.Name,
			found.schema.Version,
		)
	}
	reg.dataMap[name] = loadedDataSource{
		schema:     schema,
		DataSource: ds,
	}
	return nil
}

func (reg *runnersRegistry) registerContentProvider(
	ctx context.Context,
	name string,
	schema *plugin.Schema,
	cp *plugin.ContentProvider,
) error {
	log := appctx.Log(ctx)
	log.DebugContext(
		ctx, "Registering content provider",
		"name", name,
		"plugin", schema.Name,
		"version", schema.Version,
	)
	if found, has := reg.contentMap[name]; has {
		log.ErrorContext(
			ctx, "Content provider with the same name found in two plugins",
			"name", name,
			"plugin1", schema.Name,
			"plugin1_version", schema.Version,
			"plugin2", found.schema.Name,
			"plugin2_version", found.schema.Version,
		)
		return fmt.Errorf(
			"content provider `%s` is found in two plugins: `%s@%s` and `%s@%s`",
			name,
			schema.Name,
			schema.Version,
			found.schema.Name,
			found.schema.Version,
		)
	}
	reg.contentMap[name] = loadedContentProvider{
		schema: schema,
		ContentProvider: &plugin.ContentProvider{
			Config: cp.Config,
			Args:   cp.Args,
			Doc:    cp.Doc,
			Tags:   cp.Tags,
			ContentFunc: func(ctx context.Context, params *plugin.ProvideContentParams) (*plugin.ContentProviderResult, error) {
				return schema.ProvideContent(ctx, name, params)
			},
		},
	}
	return nil
}

func (reg *runnersRegistry) registerPublisher(
	ctx context.Context,
	name string,
	schema *plugin.Schema,
	pub *plugin.Publisher,
) error {
	log := appctx.Log(ctx)
	log.DebugContext(
		ctx, "Registering publisher",
		"name", name,
		"plugin", schema.Name,
		"version", schema.Version,
	)
	if found, has := reg.publisherMap[name]; has {
		log.ErrorContext(
			ctx, "Publisher with the same name found in two plugins",
			"name", name,
			"plugin1", schema.Name,
			"plugin1_version", schema.Version,
			"plugin2", found.schema.Name,
			"plugin2_version", found.schema.Version,
		)
		return fmt.Errorf(
			"publisher `%s` is found in two plugins: `%s@%s` and `%s@%s`",
			name,
			schema.Name,
			schema.Version,
			found.schema.Name,
			found.schema.Version,
		)
	}
	reg.publisherMap[name] = loadedPublisher{
		schema:    schema,
		Publisher: pub,
	}
	return nil
}

func (reg *runnersRegistry) registerFormatter(
	ctx context.Context,
	name string,
	schema *plugin.Schema,
	frmt *plugin.Formatter,
) error {
	log := appctx.Log(ctx)
	log.DebugContext(
		ctx,
		"Registering formatter",
		"name", name,
		"plugin", schema.Name,
		"version", schema.Version,
	)

	if found, ok := reg.publisherMap[name]; ok {
		log.ErrorContext(
			ctx, "Formatter with the same name found in two plugins",
			"name", name,
			"plugin1", schema.Name,
			"plugin1_version", schema.Version,
			"plugin2", found.schema.Name,
			"plugin2_version", found.schema.Version,
		)
		return fmt.Errorf(
			"formatter `%s` is found in two plugins: `%s@%s` and `%s@%s`",
			name,
			schema.Name,
			schema.Version,
			found.schema.Name,
			found.schema.Version,
		)
	}
	reg.formatterMap[name] = loadedFormatter{
		schema:    schema,
		Formatter: frmt,
	}
	return nil
}

func (reg *runnersRegistry) loadBinary(ctx context.Context, name, binaryPath string) (err error) {
	tracer := appctx.Tracer(ctx)
	log := appctx.Log(ctx)
	log = log.With("plugin", name, "binary_path", binaryPath)

	ctx, span := tracer.Start(ctx, "runnersRegistry.loadBinary", trace.WithAttributes(
		attribute.String("name", name),
	))

	log.InfoContext(ctx, "Loading a plugin")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "success")
		}
		span.End()
	}()

	if info, err := os.Stat(binaryPath); os.IsNotExist(err) {
		log.ErrorContext(ctx, "The binary for plugin not found", "err", err)
		return err
	} else if info.IsDir() {
		log.ErrorContext(ctx, "The path to a plugin binary points to a directory")
		return errors.New("plugin binary path is a directory")
	}
	p, close, err := pluginapiv1.NewClient(name, binaryPath, log)
	if err != nil {
		log.ErrorContext(ctx, "Error creating a new plugin client", "err", err)
	}
	return reg.registerPlugin(ctx, p, close)
}

func (reg *runnersRegistry) closeAll(ctx context.Context) error {
	log := appctx.Log(ctx)

	var errs error
	for _, p := range reg.pluginMap {
		if err := p.closefn(); err != nil {
			log.ErrorContext(ctx, "Failed to close a plugin", "name", p.Name, "version", p.Version, "err", err)
			errs = errors.Join(err, fmt.Errorf("failed to close a plugin: %s", p.Name))
		}
	}
	return errs
}
