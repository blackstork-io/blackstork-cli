// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package engine

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/maps"

	"github.com/blackstork-io/blackstork-cli/eval"
	"github.com/blackstork-io/blackstork-cli/parser"
	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugin/resolver"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

// Engine is the main entry point for the main workflow. It is responsible for installing plugins,
// parsing, loading and evaluating the template files, and fetching data.
type Engine struct {
	builtin *plugin.Schema
	config  *definitions.GlobalConfig

	blocks  parser.BlocksRegistry
	runners eval.RunnersRegistry

	lockFile  *resolver.LockFile
	resolver  *resolver.Resolver
	fileMap   map[string]*hcl.File
	env       plugindata.Map
	sourceDir string
}

// New creates a new Engine instance with the provided options.
func New(options ...Option) *Engine {
	opts := defaultOptions
	for _, opt := range options {
		opt(&opts)
	}
	return &Engine{
		builtin: opts.builtin,
		config: &definitions.GlobalConfig{
			PluginRegistry: &definitions.PluginRegistry{
				BaseURL:   opts.registryBaseURL,
				MirrorDir: "",
			},
			CacheDir:       opts.cacheDir,
			EnvVarsPattern: definitions.DefaultEnvVarsPattern,
		},
	}
}

func (e *Engine) PluginResolver() *resolver.Resolver {
	return e.resolver
}

// func (e *Engine) PluginRunner() *runners.RunnersRegistry {
// 	return e.runners
// }

func (e *Engine) LockFile() *resolver.LockFile {
	return e.lockFile
}

func (e *Engine) FileMap() map[string]*hcl.File {
	return e.fileMap
}

func (e *Engine) Blocks() parser.BlocksRegistry {
	return e.blocks
}

func (e *Engine) Install(ctx context.Context, upgrade bool) (diags diagnostics.Diag) {
	tracer := appctx.Tracer(ctx)
	log := appctx.Log(ctx)

	ctx, span := tracer.Start(ctx, "Engine.Install", trace.WithAttributes(
		attribute.Bool("upgrade", upgrade),
	))
	log.InfoContext(ctx, "Installing plugins", "upgrade", upgrade)
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	if e.resolver == nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Plugin resolver is not loaded",
			Detail:   "Load plugin resolver before installing",
		})
		return diags
	}
	lockFile, diag := e.resolver.Install(ctx, e.lockFile, upgrade)
	if diags.Extend(diag) {
		return diags
	}
	e.lockFile = lockFile
	err := resolver.SaveLockFileTo(path.Join(e.sourceDir, defaultLockFile), lockFile)
	if err != nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to save lock file",
			Detail:   err.Error(),
		})
	}

	return diags
}

func (e *Engine) ParseDir(ctx context.Context, sourceDir string) (diags diagnostics.Diag) {
	e.sourceDir = sourceDir

	return e.ParseDirFS(ctx, os.DirFS(sourceDir))
}

func (e *Engine) ParseDirFS(ctx context.Context, sourceDir fs.FS) (diags diagnostics.Diag) {
	tracer := appctx.Tracer(ctx)
	log := appctx.Log(ctx)

	ctx, span := tracer.Start(ctx, "Engine.ParseDir")
	log.InfoContext(ctx, "Parsing templates in the directory", "dir_path", sourceDir)
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	e.blocks, e.fileMap, diags = parser.ParseDir(ctx, log, sourceDir)
	if diags.HasErrors() {
		return diags
	}
	if e.blocks != nil && e.blocks.GetGlobalConfig() != nil {
		globalConfig := e.blocks.GetGlobalConfig()
		cfg, diag := globalConfig.Parse(ctx)
		if !diags.Extend(diag) {
			e.config.Merge(cfg)
		}
	}

	log.InfoContext(
		ctx, "Blocks found in the templates",
		"dir_path", sourceDir,
		"global_config_found", e.blocks.GetGlobalConfig() != nil,
		"config_blocks", len(e.blocks.GetConfigDefsMap()),
		"sections", len(e.blocks.GetSectionDefsMap()),
		"exec_blocks", len(e.blocks.GetExecBlockDefsMap()),
		"documents", len(e.blocks.GetDocumentDefsMap()),
	)

	return diags
}

func (e *Engine) Lint(ctx context.Context, fullLint bool) (diags diagnostics.Diag) {
	tracer := appctx.Tracer(ctx)
	log := appctx.Log(ctx)

	ctx, span := tracer.Start(ctx, "Engine.Lint", trace.WithAttributes(
		attribute.Bool("fullLint", fullLint),
	))
	log.InfoContext(ctx, "Linting all documents", "full_lint", fullLint)

	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	for _, doc := range e.blocks.GetDocumentDefsMap() {
		log.DebugContext(ctx, "Linting document", "document", doc.Name)
		docParsed, diag := parser.ParseDocument(ctx, e.blocks, doc)
		diags.Extend(diag)
		if fullLint {
			emptyDataCtx := plugindata.Map{}
			_, diag = eval.LoadDocument(ctx, e.runners, docParsed, emptyDataCtx)
			diags.Extend(diag)
		}
	}
	return diags
}

func (e *Engine) LoadPluginResolver(ctx context.Context, includeRemote bool) (diags diagnostics.Diag) {
	tracer := appctx.Tracer(ctx)
	log := appctx.Log(ctx)

	ctx, span := tracer.Start(ctx, "Engine.LoadPluginResolver", trace.WithAttributes(
		attribute.String("includeRemote", fmt.Sprint(includeRemote)),
	))

	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	pluginDir := filepath.Join(e.config.CacheDir, "plugins")
	// Adding a cache dir plugins as a local source
	sources := []resolver.Source{
		resolver.NewLocal(pluginDir),
	}

	log.DebugContext(
		ctx,
		"Loading a plugin resolver",
		"include_remote", includeRemote,
		"plugins_dir", string(pluginDir),
	)

	if e.config.PluginRegistry != nil {
		if e.config.PluginRegistry.MirrorDir != "" {
			mirrorDirInfo, err := os.Stat(e.config.PluginRegistry.MirrorDir)
			if err != nil || !mirrorDirInfo.IsDir() {
				return diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Can't find a registry mirror directory",
					Detail: fmt.Sprintf(
						"Can't find a directory specified as a registry mirror: `%s`",
						e.config.PluginRegistry.MirrorDir,
					),
				}}
			}
			sources = append(sources, resolver.NewLocal(e.config.PluginRegistry.MirrorDir))
		}
		if includeRemote && e.config.PluginRegistry.BaseURL != "" {
			sources = append(sources, resolver.NewRemote(resolver.RemoteOptions{
				BaseURL:     e.config.PluginRegistry.BaseURL,
				DownloadDir: pluginDir,
				UserAgent:   fmt.Sprintf("blackstork-cli/%s", "version"),
			}))
		}
	}
	var err error
	e.lockFile, err = resolver.ReadLockFileFrom(path.Join(e.sourceDir, defaultLockFile))
	if err != nil {
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to read a lock file",
			Detail:   err.Error(),
		}}
	}
	resolve, diags := resolver.NewResolver(e.config.PluginVersions, sources)
	e.resolver = resolve
	return diags
}

func (e *Engine) LoadPluginRunner(ctx context.Context) (diags diagnostics.Diag) {
	tracer := appctx.Tracer(ctx)
	log := appctx.Log(ctx)

	ctx, span := tracer.Start(ctx, "Engine.LoadPluginRunner")
	log.DebugContext(ctx, "Loading a plugin runner")
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()

	binaryMap, diag := e.resolver.Resolve(ctx, e.lockFile)
	if diags.Extend(diag) {
		return diags
	}
	var err error
	e.runners, err = eval.LoadRunnersRegistry(ctx, binaryMap, e.builtin)
	if err != nil {
		diags.Extend(diagnostics.Diag{
			{
				Severity: hcl.DiagError,
				Summary:  err.Error(),
			},
		})
		return diags
	}
	return diags
}

func (e *Engine) PrintDiagnostics(output io.Writer, diags diagnostics.Diag, colorize bool) {
	diagnostics.PrintDiags(output, diags, e.fileMap, colorize)
}

func (e *Engine) loadStandaloneDataBlock(
	ctx context.Context,
	source, name string,
) (_ *eval.PluginDataAction, diags diagnostics.Diag) {
	tracer := appctx.Tracer(ctx)
	log := appctx.Log(ctx)

	ctx, span := tracer.Start(ctx, "Engine.loadGlobalData", trace.WithAttributes(
		attribute.String("data_source", source),
		attribute.String("name", name),
	))
	log.InfoContext(ctx, "Loading a standalone data block", "data_source", source, "name", name)
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	if e.blocks == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "No blocks registered",
			Detail:   "No template blocks found registerd in the endinge",
		}}
	}
	if e.runners == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Plugin runners are not registered",
			Detail:   "Register the runners registry before evaluating",
		}}
	}
	data, ok := e.blocks.GetExecBlockDefByKey(definitions.Key{
		Kind:   definitions.BlockKindData,
		Runner: source,
		Name:   name,
	})
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Data source is not found",
			Detail:   fmt.Sprintf("Data source named `%s` not found in installed plugins", name),
		}}
	}
	parsedData, diag := parser.ParseDataBlock(ctx, e.blocks, data, nil)
	if diags.Extend(diag) {
		return nil, diags
	}
	emptyDataCtx := plugindata.Map{}
	loadedData, diag := eval.LoadDataAction(ctx, e.runners, parsedData, emptyDataCtx)
	if diags.Extend(diag) {
		return nil, diags
	}
	return loadedData, diags
}

func (e *Engine) loadDocumentData(
	ctx context.Context,
	doc string,
	path []string,
) (_ plugindata.Data, diags diagnostics.Diag) {
	tracer := appctx.Tracer(ctx)
	log := appctx.Log(ctx)

	pathStr := strings.Join(path, ".")

	ctx, span := tracer.Start(ctx, "Engine.loadDocumentData", trace.WithAttributes(
		attribute.String("document", doc),
		attribute.String("data_path", pathStr),
	))
	log.InfoContext(ctx, "Loading document data", "document", doc, "data_path", path)
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	if e.blocks == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "No files parsed",
			Detail:   "Parse files before selecting a path",
		}}
	}
	if e.runners == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Plugin runners are not loaded",
			Detail:   "Load plugin runners before evaluating the template",
		}}
	}
	docBlock, ok := e.blocks.GetDocumentDefByName(doc)
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Document not found",
			Detail:   fmt.Sprintf("Document template `%s` not found", doc),
		}}
	}
	docParsed, diag := parser.ParseDocument(ctx, e.blocks, docBlock)
	if diags.Extend(diag) {
		return nil, diags
	}

	dataCtx := plugindata.Map{}
	inputs, diag := LoadInputs(ctx, docParsed)
	if diags.Extend(diag) {
		return nil, diags
	}
	// settings inputs in the context
	dataCtx[plugin.InputsDataKey] = inputs

	// settings inputs in the eval context for HCL
	evalCtx := appctx.GetEvalContext(ctx)
	evalCtx.Variables[plugin.InputsDataKey] = plugindata.PluginDataToCty(inputs)

	document, diag := eval.LoadDocument(ctx, e.runners, docParsed, dataCtx)
	if diags.Extend(diag) {
		return nil, diags
	}

	data, diag := document.FetchDataWithPath(ctx, path)
	if diags.Extend(diag) {
		return nil, diags
	}
	return data, diags
}

var ErrInvalidDataTarget = diagnostics.Diag{{
	Severity: hcl.DiagError,
	Summary:  "Invalid data target",
	Detail:   "Target must be in the format 'document.<doc-name>.data.<plugin-name>.<block-name>' or 'data.<plugin-name>.<block-name>'",
}}

func (e *Engine) FetchData(ctx context.Context, target string) (result plugindata.Data, diags diagnostics.Diag) {
	tracer := appctx.Tracer(ctx)
	log := appctx.Log(ctx)

	log.InfoContext(ctx, "Fetching data")

	ctx, span := tracer.Start(ctx, "Engine.FetchData", trace.WithAttributes(
		attribute.String("target", target),
	))
	log.InfoContext(ctx, "Fetching the data", "target", target)
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	head, base, ok := strings.Cut(target, ".")
	if !ok {
		return nil, ErrInvalidDataTarget
	}
	var loadedData *eval.PluginDataAction
	var diag diagnostics.Diag
	switch head {
	case "document":
		// Possible options:
		// - `<document-name>.data`
		// - `<document-name>.data.<data-source>`
		// - `<document-name>.data.<data-source>.<block-name>`
		parts := strings.Split(base, ".")
		// At the minimum, `<document-name>.data`
		if len(parts) < 2 {
			return nil, ErrInvalidDataTarget
		}
		if parts[1] != "data" {
			return nil, ErrInvalidDataTarget
		}
		docName := parts[0]
		path := parts[2:]

		result, diag = e.loadDocumentData(ctx, docName, path)
		// Wrap the result, to have `data` root key
		result = plugindata.Map{"data": result}

	case "data":
		parts := strings.Split(base, ".")
		if len(parts) != 2 {
			return nil, ErrInvalidDataTarget
		}
		loadedData, diag = e.loadStandaloneDataBlock(ctx, parts[0], parts[1])
		if diags.Extend(diag) {
			return nil, diags
		}
		result, diag = loadedData.FetchData(ctx)
	default:
		return nil, ErrInvalidDataTarget
	}
	if diags.Extend(diag) {
		return nil, diags
	}

	return result, nil
}

func (e *Engine) loadEnv(ctx context.Context) (envMap plugindata.Map, diags diagnostics.Diag) {
	log := appctx.Log(ctx)
	log.DebugContext(ctx, "Loading env vars")
	envMap = plugindata.Map{}
	if e.config == nil || e.config.EnvVarsPattern == nil {
		return envMap, diags
	}
	evalCtx := utils.EvalContextByVar(appctx.GetEvalContext(ctx), "env")
	if evalCtx == nil {
		return envMap, diags
	}
	for k, v := range evalCtx.Variables["env"].AsValueMap() {
		if !e.config.EnvVarsPattern.Match(k) {
			continue
		}
		envMap[k] = plugindata.String(v.AsString())
	}
	log.DebugContext(ctx, "Env vars loaded", "env_vars", maps.Keys(envMap))
	return envMap, diags
}

func (e *Engine) initialDataCtx(ctx context.Context) (data plugindata.Map, diags diagnostics.Diag) {
	if e.env == nil {
		e.env, diags = e.loadEnv(ctx)
	}
	data = plugindata.Map{
		"env": e.env,
	}
	return data, diags
}

func (e *Engine) RenderContent(
	ctx context.Context,
	target string,
	requiredTags []string,
) (doc *eval.Document, content *plugin.ContentSection, data plugindata.Map, diags diagnostics.Diag) {
	tracer := appctx.Tracer(ctx)
	log := appctx.Log(ctx)
	log = log.With("document", target)
	ctx = appctx.WithLog(ctx, log)

	ctx, span := tracer.Start(ctx, "Engine.RenderContent", trace.WithAttributes(
		attribute.String("target", target),
	))
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()

	log.InfoContext(ctx, "Parsing template")

	docParsed, diag := e.parseDocument(ctx, target)
	if diags.Extend(diag) {
		return nil, nil, nil, diags
	}

	dataCtx, diag := e.initialDataCtx(ctx)
	if diags.Extend(diag) {
		return nil, nil, nil, diags
	}

	log.InfoContext(ctx, "Loading inputs")

	inputs, diag := LoadInputs(ctx, docParsed)
	if diags.Extend(diag) {
		return nil, nil, nil, diags
	}
	// settings inputs in the context
	dataCtx[plugin.InputsDataKey] = inputs

	// settings inputs in the eval context for HCL
	evalCtx := appctx.GetEvalContext(ctx)
	evalCtx.Variables[plugin.InputsDataKey] = plugindata.PluginDataToCty(inputs)

	log.InfoContext(ctx, "Loading document")

	doc, diag = e.loadDocument(ctx, docParsed, dataCtx)
	if diags.Extend(diag) {
		return nil, nil, nil, diags
	}

	subData := appctx.SubstituteData(ctx)
	if subData == nil {
		log.InfoContext(ctx, "Fetching data")

		data, diag = doc.FetchData(ctx)
		if diags.Extend(diag) {
			return nil, nil, nil, diags
		}
	} else {
		log.InfoContext(ctx, "Substitute data found, skipping data block execution")
		data = subData
	}

	dataCtx[definitions.BlockKindData] = data

	log.InfoContext(ctx, "Rendering content")

	content, data, diag = doc.RenderContent(ctx, dataCtx, requiredTags)
	if diags.Extend(diag) {
		return nil, nil, nil, diags
	}

	return doc, content, data, diags
}

func (e *Engine) PublishContent(
	ctx context.Context,
	doc *eval.Document,
	content *plugin.ContentSection,
	data plugindata.Map,
	executePublishBlocks bool,
) (diags diagnostics.Diag) {
	tracer := appctx.Tracer(ctx)
	log := appctx.Log(ctx)

	documentTemplateName := doc.GetTemplateName()

	ctx, span := tracer.Start(ctx, "Engine.Publish", trace.WithAttributes(
		attribute.String("document", documentTemplateName),
	))
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	log = log.With("document", documentTemplateName)

	// If publishing is not requested, execute the default publisher / formatter
	if !executePublishBlocks {
		log.DebugContext(
			ctx, "Executing a default publisher",
			"publisher", doc.DefaultPublish.RunnerName,
		)
		var publisher *eval.PluginPublishAction
		for _, p := range doc.PublishBlocks {
			if p.RunnerName == doc.DefaultPublish.RunnerName {
				publisher = p
				break
			}
		}
		if publisher == nil {
			// No registered publisher of a default expected runner found,
			// so we're registering the new one
			doc.PublishBlocks = []*eval.PluginPublishAction{doc.DefaultPublish}
		}
	}

	formattedContentMap, diag := doc.FormatContent(ctx, content, data)
	if diags.Extend(diag) {
		return diags
	}

	log.DebugContext(
		ctx, "Formatted content map prepared for publishing",
		"formatted_content_count", len(formattedContentMap),
		"content_formats", maps.Keys(formattedContentMap),
	)

	log.InfoContext(ctx, "Publishing the document")
	diag = doc.Publish(ctx, content, formattedContentMap, data)
	diags.Extend(diag)
	return diags
}

func (e *Engine) parseDocument(ctx context.Context, name string) (_ *definitions.Document, diags diagnostics.Diag) {
	tracer := appctx.Tracer(ctx)
	log := appctx.Log(ctx)

	ctx, span := tracer.Start(ctx, "Engine.loadDocument", trace.WithAttributes(
		attribute.String("target", name),
	))
	log.InfoContext(ctx, "Fetching document template")
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	if e.runners == nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Plugin runner is not loaded",
			Detail:   "Plugin runner must be loaded before loading a document template",
		})
		return nil, diags
	}
	doc, ok := e.blocks.GetDocumentDefByName(name)
	if !ok {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Document not found",
			Detail:   fmt.Sprintf("Document template `%s` not found", name),
		})
		return nil, diags
	}
	log.InfoContext(ctx, "Parsing document template")
	docParsed, diag := parser.ParseDocument(ctx, e.blocks, doc)
	if diags.Extend(diag) {
		log.ErrorContext(ctx, "Error while parsing template", "err", diagnostics.GetDiagsDetails(diags))
		return nil, diags
	}
	return docParsed, diags
}

func (e *Engine) loadDocument(ctx context.Context, doc *definitions.Document, dataCtx plugindata.Map) (_ *eval.Document, diags diagnostics.Diag) {
	log := appctx.Log(ctx)
	log.InfoContext(ctx, "Loading document template")
	docLoaded, diag := eval.LoadDocument(ctx, e.runners, doc, dataCtx)
	if diags.Extend(diag) {
		log.ErrorContext(ctx, "Error while loading template", "err", diagnostics.GetDiagsDetails(diags))
		return nil, diags
	}
	return docLoaded, diags
}

func (e *Engine) Cleanup() error {
	if e.runners != nil {
		return e.runners.CloseAll()
	}
	return nil
}
