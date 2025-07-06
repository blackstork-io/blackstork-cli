package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/blackstork-io/fabric/internal/engine"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/plugins/builtin"
)

var fullLint bool

func init() {
	lintCmd.Flags().BoolVar(&fullLint, "full", false, "Lint plugin bodies (requires plugins to be installed)")
	rootCmd.AddCommand(lintCmd)
}

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Evaluate *.fabric files for syntax mistakes",
	Long:  `Doesn't call plugins, only checks the *.fabric templates for correctness`,
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
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
		if fullLint {
			if diags.Extend(eng.LoadPluginResolver(ctx, false)) {
				return
			}
			if diags.Extend(eng.LoadPluginRunner(ctx)) {
				return
			}
		}
		diag = eng.Lint(ctx, fullLint)
		diags.Extend(diag)
		return
	},
}
