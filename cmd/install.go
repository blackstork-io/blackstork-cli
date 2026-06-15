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

var installUpgrade bool

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVarP(&installUpgrade, "upgrade", "u", false, "Upgrade plugin versions")
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install plugins",
	Long:  "Install Fabric plugins",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
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
		diag = eng.LoadPluginResolver(ctx, true)
		if diags.Extend(diag) {
			return err
		}
		diag = eng.Install(ctx, installUpgrade)
		if diags.Extend(diag) {
			return err
		}
		return err
	},
}
