package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"

	"github.com/blackstork-io/fabric/engine"
	"github.com/blackstork-io/fabric/internal/builtin"
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/utils"
	"github.com/blackstork-io/fabric/print"
	"github.com/blackstork-io/fabric/print/htmlprint"
	"github.com/blackstork-io/fabric/print/mdprint"
)

var (
	publish bool
	format  string
	tags    string
)

func init() {
	rootCmd.AddCommand(renderCmd)
	renderCmd.Flags().BoolVar(&publish, "publish", false, "publish the rendered document")
	renderCmd.Flags().StringVar(&format, "format", "md", "default output format of the document (md, html or pdf)")
	renderCmd.Flags().
		StringVar(&tags, "with-meta-tags", "", "comma separated list of meta tags. Only content blocks matching these tags will be rendered")

	renderCmd.SetUsageTemplate(UsageTemplate(
		[2]string{"TARGET", "name of the document to be rendered as 'document.<name>'"},
	))
}

var renderCmd = &cobra.Command{
	Use:   "render TARGET",
	Short: "Render the document",
	Long:  `Render the specified document and either publish it or output it to stdout.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		target := strings.TrimSpace(args[0])
		const docPrefix = definitions.BlockKindDocument + "."
		switch {
		case strings.HasPrefix(target, docPrefix):
			target = target[len(docPrefix):]
		default:
			return fmt.Errorf("target should have the format '%s<name_of_the_document>'", docPrefix)
		}
		requiredTags := slices.DeleteFunc(
			utils.FnMap(
				strings.Split(tags, ","),
				strings.TrimSpace,
			),
			func(tag string) bool { return tag == "" },
		)

		logger := slog.Default()

		ctx := cmd.Context()
		ctx = utils.WithLog(ctx, logger)

		var diags diagnostics.Diag
		eng := engine.New(
			engine.WithLogger(logger),
			engine.WithTracer(tracer),
			engine.WithBuiltIn(builtin.Plugin(version, slog.Default(), tracer)),
		)
		defer func() {
			err = exitCommand(eng, cmd, diags)
		}()
		diag := eng.ParseDir(ctx, cliArgs.sourceDir)
		if diags.Extend(diag) {
			return
		}
		diag = eng.LoadPluginResolver(ctx, false)
		if diags.Extend(diag) {
			return
		}
		diag = eng.LoadPluginRunner(ctx)
		if diags.Extend(diag) {
			return
		}

		doc, content, dataCtx, diag := eng.RenderContent(ctx, target, requiredTags)
		if diags.Extend(diag) {
			return
		}

		if publish {
			diag = eng.PublishContent(ctx, target, doc, content, dataCtx)
			if diags.Extend(diag) {
				return
			}
			// Do not print to stdout if publishing is requested
			return
		}

		logger.InfoContext(ctx, "Printing to stdout", "format", format)

		var printer print.Printer
		switch format {
		case "md":
			printer = mdprint.New()
		case "html":
			printer = htmlprint.New()
		default:
			diags.Add("Unsupported format", fmt.Sprintf("Format '%s' is not supported for stdout", format))
			return
		}
		printer = print.WithLogging(printer, slog.Default(), slog.String("format", format))
		printer = print.WithTracing(printer, tracer, attribute.String("format", format))
		err = printer.Print(ctx, os.Stdout, content)
		if err != nil {
			diags.AppendErr(err, "Error while printing")
		}

		// Making sure the stdout printout has a linebreak at the end
		fmt.Printf("\n")

		return nil
	},
}
