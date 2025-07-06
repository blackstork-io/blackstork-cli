package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/blackstork-io/fabric/internal/engine"
	"github.com/blackstork-io/fabric/plugins/builtin"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
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
			engine.WithLogger(slog.Default()),
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
		diag = eng.LoadPluginResolver(ctx, true)
		if diags.Extend(diag) {
			return
		}
		diag = eng.Install(ctx, installUpgrade)
		if diags.Extend(diag) {
			return
		}
		return
	},
}
