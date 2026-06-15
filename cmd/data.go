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
	"encoding/json"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/blackstork-io/blackstork-cli/engine"
	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugins/builtin"
)

func init() {
	rootCmd.AddCommand(dataCmd)
	dataCmd.SetUsageTemplate(UsageTemplate(
		[2]string{
			"PATH",
			"a path to data blocks to be executed. The path format is 'document.<doc-name>.data[.<plugin-name>[.<data-name>]]'.",
		},
	))
}

var dataCmd = &cobra.Command{
	Use:   "data TARGET",
	Short: "Execute the data blocks that match the path",
	Long:  `Execute the data blocks that match the path and print out prettified JSON to stdout`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()

		log := slog.Default()

		ctx = appctx.WithTracer(ctx, tracer)
		ctx = appctx.WithLog(ctx, log)

		var diags diagnostics.Diag
		eng := engine.New(
			engine.WithBuiltIn(builtin.Plugin(version, log, tracer)),
		)
		defer func() {
			err = exitCommand(eng, cmd, diags)
		}()
		diag := eng.ParseDir(ctx, cliArgs.sourceDir)
		if diags.Extend(diag) {
			log.ErrorContext(ctx, "Error while parsing directory", "err", diags.Error())
			return err
		}
		diag = eng.LoadPluginResolver(ctx, false)
		if diags.Extend(diag) {
			log.ErrorContext(ctx, "Error while loading plugin resolver", "err", diags.Error())
			return err
		}
		diags = eng.LoadPluginRunner(ctx)
		if diags.Extend(diag) {
			log.ErrorContext(ctx, "Error while loading plugin runner", "err", diags.Error())
			return err
		}
		res, diag := eng.FetchData(ctx, args[0])
		if diags.Extend(diag) {
			log.ErrorContext(ctx, "Error while fetching data", "err", diags.Error())
			return err
		}
		val := res.Any()
		ser, err := json.MarshalIndent(val, "", "    ")
		if diags.AppendErr(err, "Failed to serialize data output to JSON") {
			log.ErrorContext(ctx, "Error while serializing to JSON", "err", diags.Error())
			return err
		}
		_, err = os.Stdout.Write(ser)

		diags.AppendErr(err, "Failed to output json data")
		return err
	},
}
