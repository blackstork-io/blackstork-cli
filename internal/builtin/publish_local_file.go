package builtin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

func makeLocalFilePublisher(log *slog.Logger, tracer trace.Tracer) *plugin.Publisher {
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if tracer == nil {
		tracer = nooptrace.Tracer{}
	}
	return &plugin.Publisher{
		Doc:  "Publishes content to local file",
		Tags: []string{},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "path",
					Doc:         "Path to the file",
					Type:        cty.String,
					ExampleVal:  cty.StringVal("dist/output.md"),
					Constraints: constraint.Required,
				},
			},
		},
		Formats:     []string{"md", "pdf", "html"},
		PublishFunc: publishLocalFile(log, tracer),
	}
}

func publishLocalFile(log *slog.Logger, tracer trace.Tracer) plugin.PublishFunc {
	return func(ctx context.Context, params *plugin.PublishParams) diagnostics.Diag {
		var document *plugin.ContentSection
		//content, _ := getDocument(params.DataContext)
		//document, _ := parseScope(params.DataContext)
		if document == nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse document",
				Detail:   "document is required",
			}}
		}

		if params.FormattedContent == nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "No formatted content received",
				Detail:   "formatted content is required",
			}}
		}

		datactx := params.DataContext
		// datactx["formatted_content"] = plugindata.String(params.FormattedContent.Content)

		content := params.FormattedContent.Content
		format := params.FormattedContent.Format

		log.InfoContext(ctx, "PUBLISHING A LOCAL FILE", "format", format)

		// var printer print.Printer = mdprint.New()
		// switch params.Format {
		// case plugin.OutputFormatMD:
		// 	printer = mdprint.New()
		// case plugin.OutputFormatHTML:
		// 	printer = htmlprint.New()
		// case plugin.OutputFormatPDF:
		// 	printer = pdfprint.New()
		// default:
		// 	return diagnostics.Diag{{
		// 		Severity: hcl.DiagError,
		// 		Summary:  "Unsupported format",
		// 		Detail:   "Only md, html and pdf formats are supported",
		// 	}}
		// }
		// printer = print.WithLogging(printer, log, slog.String("format", format))
		// printer = print.WithTracing(printer, tracer, attribute.String("format", format))

		pathAttr := params.Args.GetAttrVal("path")
		if pathAttr.IsNull() || pathAttr.AsString() == "" {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse arguments",
				Detail:   "path is required",
			}}
		}
		path, err := templatePath(pathAttr.AsString(), datactx)
		if err != nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to render a path value",
				Detail:   err.Error(),
			}}
		}
		log.InfoContext(ctx, "Writing to a file", "path", path)
		dir := filepath.Dir(path)
		err = os.MkdirAll(dir, 0o755)
		if err != nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to create a directory for the output file",
				Detail:   err.Error(),
			}}
		}
		fs, err := os.Create(path)
		if err != nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to create a file",
				Detail:   err.Error(),
			}}
		}
		defer fs.Close()

		bytesCount, err := fs.Write(content)
		// err = printer.Print(ctx, fs, document)
		if err != nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to write to a file",
				Detail:   err.Error(),
			}}
		}
		log.DebugContext(
			ctx, "Content written to a file",
			"bytes_count", bytesCount,
			"path", path,
		)
		return nil
	}
}

func templatePath(pattern string, datactx plugindata.Map) (string, error) {
	tmpl, err := template.New("pattern").Funcs(sprig.FuncMap()).Parse(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to parse a text template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, datactx.Any())
	if err != nil {
		return "", fmt.Errorf("failed to execute a text template: %w", err)
	}
	return filepath.Abs(filepath.Clean(strings.TrimSpace(buf.String())))
}
