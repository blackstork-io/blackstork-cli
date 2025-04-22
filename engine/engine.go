package engine

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/eval"
	"github.com/blackstork-io/fabric/parser"
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/utils"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/plugindata"
	"github.com/blackstork-io/fabric/plugin/resolver"
	"github.com/blackstork-io/fabric/plugin/runner"
)

// Engine is the main entry point for the fabric engine. It is responsible for
// loading and evaluating fabric files, installing plugins, and fetching data.
// It is also responsible for managing the plugin resolver and runner.
type Engine struct {
	builtin   *plugin.Schema
	logger    *slog.Logger
	tracer    trace.Tracer
	config    *definitions.GlobalConfig
	blocks    *parser.DefinedBlocks
	runner    *runner.Runner
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
		logger:  opts.logger,
		tracer:  opts.tracer,
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

func (e *Engine) PluginRunner() *runner.Runner {
	return e.runner
}

func (e *Engine) LockFile() *resolver.LockFile {
	return e.lockFile
}

func (e *Engine) FileMap() map[string]*hcl.File {
	return e.fileMap
}

func (e *Engine) ParsedBlocks() *parser.DefinedBlocks {
	return e.blocks
}

func (e *Engine) Install(ctx context.Context, upgrade bool) (diags diagnostics.Diag) {
	ctx, span := e.tracer.Start(ctx, "Engine.Install", trace.WithAttributes(
		attribute.Bool("upgrade", upgrade),
	))
	e.logger.InfoContext(ctx, "Installing plugins", "upgrade", upgrade)
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
		return
	}
	lockFile, diag := e.resolver.Install(ctx, e.lockFile, upgrade)
	if diags.Extend(diag) {
		return
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

	return
}

func (e *Engine) ParseDir(ctx context.Context, sourceDir string) (diags diagnostics.Diag) {
	e.sourceDir = sourceDir

	return e.ParseDirFS(ctx, os.DirFS(sourceDir))
}

func (e *Engine) ParseDirFS(ctx context.Context, sourceDir fs.FS) (diags diagnostics.Diag) {
	ctx, span := e.tracer.Start(ctx, "Engine.ParseDir")
	e.logger.InfoContext(ctx, "Parsing fabric files", "directory", sourceDir)
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	e.blocks, e.fileMap, diags = parser.ParseDir(sourceDir)
	if diags.HasErrors() {
		return
	}
	if e.blocks.GlobalConfig != nil {
		cfg, diag := e.blocks.GlobalConfig.Parse(ctx)
		if !diags.Extend(diag) {
			e.config.Merge(cfg)
		}
	}

	return
}

func (e *Engine) Lint(ctx context.Context, fullLint bool) (diags diagnostics.Diag) {
	ctx, span := e.tracer.Start(ctx, "Engine.Lint", trace.WithAttributes(
		attribute.Bool("fullLint", fullLint),
	))
	e.logger.InfoContext(ctx, "Linting all documents", "full_lint", fullLint)
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	for _, doc := range e.blocks.Documents {
		e.logger.DebugContext(ctx, "Linting document", "document", doc.Name)
		parsedDoc, diag := e.blocks.ParseDocument(ctx, doc)
		diags.Extend(diag)
		if fullLint {
			_, diag = eval.LoadDocument(ctx, e.runner, parsedDoc)
			diags.Extend(diag)
		}
	}
	return diags
}

func (e *Engine) LoadPluginResolver(ctx context.Context, includeRemote bool) (diags diagnostics.Diag) {
	ctx, span := e.tracer.Start(ctx, "Engine.LoadPluginResolver", trace.WithAttributes(
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
		resolver.NewLocal(pluginDir, e.logger, e.tracer),
	}

	e.logger.DebugContext(
		ctx,
		"Loading plugin resolver",
		"include_remote",
		includeRemote,
		"plugins_dir",
		string(pluginDir),
	)

	if e.config.PluginRegistry != nil {
		if e.config.PluginRegistry.MirrorDir != "" {
			mirrorDirInfo, err := os.Stat(e.config.PluginRegistry.MirrorDir)
			if err != nil || !mirrorDirInfo.IsDir() {
				return diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Can't find a mirror directory",
					Detail: fmt.Sprintf(
						"Can't find a directory specified as a mirror: %s",
						e.config.PluginRegistry.MirrorDir,
					),
				}}
			}
			sources = append(sources, resolver.NewLocal(e.config.PluginRegistry.MirrorDir, e.logger, e.tracer))
		}
		if includeRemote && e.config.PluginRegistry.BaseURL != "" {
			sources = append(sources, resolver.NewRemote(resolver.RemoteOptions{
				BaseURL:     e.config.PluginRegistry.BaseURL,
				DownloadDir: pluginDir,
				UserAgent:   fmt.Sprintf("fabric/%s", "version"),
			}))
		}
	}
	var err error
	e.lockFile, err = resolver.ReadLockFileFrom(path.Join(e.sourceDir, defaultLockFile))
	if err != nil {
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to read lock file",
			Detail:   err.Error(),
		}}
	}
	resolve, diags := resolver.NewResolver(e.config.PluginVersions,
		resolver.WithSources(sources...),
		resolver.WithLogger(e.logger),
		resolver.WithTracer(e.tracer),
	)
	e.resolver = resolve
	return diags
}

func (e *Engine) LoadPluginRunner(ctx context.Context) (diags diagnostics.Diag) {
	ctx, span := e.tracer.Start(ctx, "Engine.LoadPluginRunner")
	e.logger.DebugContext(ctx, "Loading plugin runner")
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	binaryMap, diag := e.resolver.Resolve(ctx, e.lockFile)
	if diags.Extend(diag) {
		return diag
	}
	e.runner, diag = runner.Load(ctx, binaryMap, e.builtin, e.logger, e.tracer)
	diag.Extend(diag)
	return diag
}

func (e *Engine) PrintDiagnostics(output io.Writer, diags diagnostics.Diag, colorize bool) {
	diagnostics.PrintDiags(output, diags, e.fileMap, colorize)
}

func (e *Engine) loadGlobalData(
	ctx context.Context,
	source, name string,
) (_ *eval.PluginDataAction, diags diagnostics.Diag) {
	ctx, span := e.tracer.Start(ctx, "Engine.loadGlobalData", trace.WithAttributes(
		attribute.String("data_source", source),
		attribute.String("name", name),
	))
	e.logger.InfoContext(ctx, "Loading global data", "data_source", source, "name", name)
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
			Detail:   "Parse files before selecting",
		}}
	}
	if e.runner == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Plugin runner is not loaded",
			Detail:   "Load plugin runner before evaluating",
		}}
	}
	data, ok := e.blocks.Plugins[definitions.Key{
		PluginKind: definitions.BlockKindData,
		PluginName: source,
		BlockName:  name,
	}]
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Data source not found",
			Detail:   fmt.Sprintf("Data source named '%s' not found", name),
		}}
	}
	parsedData, diag := e.blocks.ParsePlugin(ctx, data)
	if diags.Extend(diag) {
		return nil, diags
	}
	loadedData, diag := eval.LoadDataAction(ctx, e.runner, parsedData)
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
	pathStr := strings.Join(path, ".")

	ctx, span := e.tracer.Start(ctx, "Engine.loadDocumentData", trace.WithAttributes(
		attribute.String("document", doc),
		attribute.String("data_path", pathStr),
	))
	e.logger.InfoContext(ctx, "Loading document data", "document", doc, "data_path", path)
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
	if e.runner == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Plugin runner is not loaded",
			Detail:   "Load plugin runner before evaluating the template",
		}}
	}
	docBlock, ok := e.blocks.Documents[doc]
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Document not found",
			Detail:   fmt.Sprintf("Definition for document named '%s' not found", doc),
		}}
	}
	e.logger.DebugContext(ctx, "Parsing a document template")
	docParsed, diag := e.blocks.ParseDocument(ctx, docBlock)
	if diags.Extend(diag) {
		return nil, diags
	}

	document, diag := eval.LoadDocument(ctx, e.runner, docParsed)
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
	ctx, span := e.tracer.Start(ctx, "Engine.FetchData", trace.WithAttributes(
		attribute.String("target", target),
	))
	e.logger.InfoContext(ctx, "Fetching the data", "target", target)
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
		loadedData, diag = e.loadGlobalData(ctx, parts[0], parts[1])
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
	e.logger.DebugContext(ctx, "Loading env variables")
	envMap = plugindata.Map{}
	if e.config == nil || e.config.EnvVarsPattern == nil {
		return
	}
	evalCtx := utils.EvalContextByVar(fabctx.GetEvalContext(ctx), "env")
	if evalCtx == nil {
		return
	}
	for k, v := range evalCtx.Variables["env"].AsValueMap() {
		if !e.config.EnvVarsPattern.Match(k) {
			continue
		}
		envMap[k] = plugindata.String(v.AsString())
	}
	return
}

func (e *Engine) initialDataCtx(ctx context.Context) (data plugindata.Map, diags diagnostics.Diag) {
	if e.env == nil {
		e.env, diags = e.loadEnv(ctx)
	}
	data = plugindata.Map{
		"env": e.env,
	}
	return
}

func (e *Engine) RenderContent(
	ctx context.Context,
	target string,
	requiredTags []string,
) (doc *eval.Document, content *plugin.ContentSection, data plugindata.Data, diags diagnostics.Diag) {
	ctx, span := e.tracer.Start(ctx, "Engine.RenderContent", trace.WithAttributes(
		attribute.String("target", target),
	))
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	doc, diag := e.loadDocument(ctx, target)
	if diags.Extend(diag) {
		return nil, nil, nil, diags
	}
	dataCtx, diag := e.initialDataCtx(ctx)
	if diags.Extend(diag) {
		return
	}
	content, data, diag = doc.RenderContent(ctx, dataCtx, requiredTags)
	if diags.Extend(diag) {
		return nil, nil, nil, diags
	}

	if content.IsEmpty() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "No content to render",
			Detail:   "Document did not produce any content, perhaps it's empty or '--with-meta-tags' filter is too strict?",
			Subject:  doc.Source.Block.DefRange().Ptr(),
		})
	}
	return doc, content, data, diags
}

func (e *Engine) PublishContent(
	ctx context.Context,
	target string,
	doc *eval.Document,
	content *plugin.ContentSection,
	dataCtx plugindata.Data,
	executePublishBlocks bool,
) (diags diagnostics.Diag) {
	// FIXME: should be calling formatting somewhere here

	ctx, span := e.tracer.Start(ctx, "Engine.Publish", trace.WithAttributes(
		attribute.String("target", target),
	))
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	e.logger.InfoContext(ctx, "Publishing the content", "target", target)
	diag := doc.Publish(ctx, content, dataCtx, target, executePublishBlocks)
	diags.Extend(diag)
	return
}

func (e *Engine) loadDocument(ctx context.Context, name string) (_ *eval.Document, diags diagnostics.Diag) {
	ctx, span := e.tracer.Start(ctx, "Engine.loadDocument", trace.WithAttributes(
		attribute.String("target", name),
	))
	e.logger.InfoContext(ctx, "Loading the template", "document", name)
	defer func() {
		if diags.HasErrors() {
			span.RecordError(diags)
			span.SetStatus(codes.Error, diags.Error())
		}
		span.End()
	}()
	if e.runner == nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Plugin runner is not loaded",
			Detail:   "Load plugin runner before evaluating",
		})
		return nil, diags
	}
	doc, ok := e.blocks.Documents[name]
	if !ok {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Document not found",
			Detail:   fmt.Sprintf("Document template '%s' not found", name),
		})
		return nil, diags
	}
	parsedDoc, diag := e.blocks.ParseDocument(ctx, doc)
	if diags.Extend(diag) {
		return nil, diags
	}
	loadedDoc, diag := eval.LoadDocument(ctx, e.runner, parsedDoc)
	if diags.Extend(diag) {
		return nil, diags
	}
	return loadedDoc, diags
}

func (e *Engine) Cleanup() diagnostics.Diag {
	if e.runner != nil {
		return e.runner.Close()
	}
	return nil
}
