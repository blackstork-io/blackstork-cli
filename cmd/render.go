// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/blackstork-io/blackstork-cli/engine"
	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugins/builtin"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

var (
	publish bool
	// format  string
	tags     string
	dataFile string
)

func init() {
	rootCmd.AddCommand(renderCmd)
	renderCmd.Flags().BoolVar(&publish, "publish", false, "publish the rendered document")
	// renderCmd.Flags().StringVar(&format, "format", "md", "default output format of the document (md, html or pdf)")
	renderCmd.Flags().
		StringVar(&tags, "only-with-tags", "", "comma separated list of meta tags. Only content blocks matching these tags will be rendered")
	renderCmd.Flags().
		StringVar(&dataFile, "replace-data-with", "", "JSON file to replace data layer in the template")

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

		log := slog.Default()

		ctx := cmd.Context()
		ctx = appctx.WithLog(ctx, log)

		if dataFile != "" {
			jsonData, err := os.ReadFile(dataFile)
			if err != nil {
				log.ErrorContext(ctx, "Error while reading data from file", "file", dataFile, "err", err)
				return err
			}
			subData, err := plugindata.UnmarshalJSON(jsonData)
			if err != nil {
				log.ErrorContext(ctx, "Error unmarshaling data from file", "file", dataFile, "err", err)
				return err
			}
			subDataMap, ok := subData.(plugindata.Map)
			if !ok {
				log.ErrorContext(ctx, "Provided substitute data is not a map", "data_type", fmt.Sprintf("%T", subData))
				return errors.New("invalid substitute data type")
			}
			ctx = appctx.WithSubstituteData(ctx, subDataMap)
		}

		var diags diagnostics.Diag
		eng := engine.New(
			engine.WithBuiltIn(builtin.Plugin(version, slog.Default(), tracer)),
		)
		defer func() {
			err = exitCommand(eng, cmd, diags)
		}()
		diag := eng.ParseDir(ctx, cliArgs.sourceDir)
		if diags.Extend(diag) {
			return err
		}
		diag = eng.LoadPluginResolver(ctx, false)
		if diags.Extend(diag) {
			return err
		}
		diag = eng.LoadPluginRunner(ctx)
		if diags.Extend(diag) {
			return err
		}

		doc, content, data, diag := eng.RenderContent(ctx, target, requiredTags)
		if diags.Extend(diag) {
			return err
		}
		if content == nil || content.IsEmpty() {
			log.WarnContext(
				ctx,
				"No content produced: either no template blocks were defined or nothing was not selected for rendering",
				"doc_inputs", len(doc.Inputs),
				"doc_data_blocks", len(doc.DataBlocks),
				"doc_root_content_branches", len(doc.ContentTreeBlocks),
			)
			return nil
		}

		diag = eng.PublishContent(ctx, doc, content, data, publish)
		if diags.Extend(diag) {
			return err
		}

		if len(diags) > 0 {
			eng.PrintDiagnostics(os.Stderr, diags, !cliArgs.noColor)
		}
		// Do not print to stdout if publishing is requested

		// logger.InfoContext(ctx, "Printing to stdout", "format", format)
		//
		// var printer print.Printer
		// switch format {
		// case "md":
		// 	printer = mdprint.New()
		// case "html":
		// 	printer = htmlprint.New()
		// default:
		// 	diags.Add("Unsupported format", fmt.Sprintf("Format '%s' is not supported for stdout", format))
		// 	return
		// }
		// printer = print.WithLogging(printer, slog.Default(), slog.String("format", format))
		// printer = print.WithTracing(printer, tracer, attribute.String("format", format))
		// err = printer.Print(ctx, os.Stdout, content)
		// if err != nil {
		// 	diags.AppendErr(err, "Error while printing")
		// }
		//
		// // Making sure the stdout printout has a linebreak at the end
		// fmt.Printf("\n")

		return nil
	},
}
