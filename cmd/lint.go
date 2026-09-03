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
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/blackstork-io/blackstork-cli/engine"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugins/builtin"
)

var fullLint bool

func init() {
	lintCmd.Flags().BoolVar(&fullLint, "full", false, "validate plugin block bodies against installed runner schemas")
	rootCmd.AddCommand(lintCmd)
}

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Validate BlackStork templates",
	Long:  "Check BlackStork templates for syntax and structural errors without executing plugins. Use --full to also validate plugin block bodies against installed runner schemas.",
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ctx := cmd.Context()
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
		if fullLint {
			if diags.Extend(eng.LoadPluginResolver(ctx, false)) {
				return err
			}
			err := eng.LoadPluginRunner(ctx)
			if err != nil {
				return err
			}
		}
		diag = eng.Lint(ctx, fullLint)
		diags.Extend(diag)
		return err
	},
}
