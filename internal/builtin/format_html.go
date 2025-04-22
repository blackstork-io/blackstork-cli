package builtin

import (
	"context"
	"io"
	"log/slog"

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

func makeHTMLFormatter(logger *slog.Logger, tracer trace.Tracer) *plugin.Formatter {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if tracer == nil {
		tracer = nooptrace.Tracer{}
	}
	return &plugin.Formatter{
		Doc:     "Formats content in HTML",
		Format:  "html",
		FileExt: "html",
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "page_title",
					Doc:  "HTML Page title",
					Type: cty.String,
				},
				{
					Name:        "template_per_type",
					Doc:         "HTML templates for specific block types (content and section)",
					Type:        plugindata.Encapsulated.CtyType(),
					Constraints: constraint.RequiredMeaningful,
					ExampleVal: cty.ObjectVal(map[string]cty.Value{
						"content.text":  cty.StringVal(`<span class="text-block">{{ .self.value }}</span>`),
						"content.image": cty.StringVal(`<img src="{{ .self.src }}" alt="{{ .self.alt }}" class="img-w-10" />`),
					}),
				},
				{
					Name:        "template_per_block",
					Doc:         "HTML templates for specific content and section blocks",
					Type:        plugindata.Encapsulated.CtyType(),
					Constraints: constraint.RequiredMeaningful,
					ExampleVal: cty.ObjectVal(map[string]cty.Value{
						"content.text.foo": cty.StringVal(`<span class="text-block">{{ .self.value }}</span>`),
						"section.bar":      cty.StringVal(`<h1>{{ .self.title.value }}</h1><p>{{ .self.content }}</p>`),
					}),
				},
			},
		},
		FormatFunc: makeHTMLFormatterFunc(logger, tracer),
	}
}

func makeHTMLFormatterFunc(log *slog.Logger, tracer trace.Tracer) plugin.FormatFunc {
	return func(ctx context.Context, params *plugin.FormatParams) (*plugin.FormattedContent, diagnostics.Diag) {
		document, _ := parseScope(params.DataContext)
		if document == nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse data context",
				Detail:   "document is not found",
			}}
		}
		// datactx := params.DataContext
		// datactx["format"] = plugindata.String(params.Format)

		for _, child := range document.Children {
			childData := child.AsData().(plugindata.Map)
			log.InfoContext(
				ctx, "CHILD",
				"data", childData,
			)
		}

		log.InfoContext(ctx, "HTML FORMATTER CALLED", "params", params, "document", document)
		return &plugin.FormattedContent{
			Content: []byte("HELLO FORMATTED HTML"),
			Format: params.Format,
		}, nil

		// var printer print.Printer
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
		// printer = print.WithLogging(printer, logger, slog.String("format", params.Format.String()))
		// printer = print.WithTracing(printer, tracer, attribute.String("format", params.Format.String()))
		// pathAttr := params.Args.GetAttrVal("path")
		// if pathAttr.IsNull() || pathAttr.AsString() == "" {
		// 	return diagnostics.Diag{{
		// 		Severity: hcl.DiagError,
		// 		Summary:  "Failed to parse arguments",
		// 		Detail:   "path is required",
		// 	}}
		// }
		// path, err := templatePath(pathAttr.AsString(), datactx)
		// if err != nil {
		// 	return diagnostics.Diag{{
		// 		Severity: hcl.DiagError,
		// 		Summary:  "Failed to render a path value",
		// 		Detail:   err.Error(),
		// 	}}
		// }
		// logger.InfoContext(ctx, "Writing to a file", "path", path)
		// dir := filepath.Dir(path)
		// err = os.MkdirAll(dir, 0o755)
		// if err != nil {
		// 	return diagnostics.Diag{{
		// 		Severity: hcl.DiagError,
		// 		Summary:  "Failed to create a directory",
		// 		Detail:   err.Error(),
		// 	}}
		// }
		// fs, err := os.Create(path)
		// if err != nil {
		// 	return diagnostics.Diag{{
		// 		Severity: hcl.DiagError,
		// 		Summary:  "Failed to create a file",
		// 		Detail:   err.Error(),
		// 	}}
		// }
		// defer fs.Close()
		// err = printer.Print(ctx, fs, document)
		// if err != nil {
		// 	return diagnostics.Diag{{
		// 		Severity: hcl.DiagError,
		// 		Summary:  "Failed to write to a file",
		// 		Detail:   err.Error(),
		// 	}}
		// }
		// return nil
	}
}
