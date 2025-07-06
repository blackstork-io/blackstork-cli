package cmd

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/blackstork-io/fabric/internal/engine"
	"github.com/blackstork-io/fabric/plugins/builtin"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
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
		if diags.Extend(eng.LoadPluginResolver(ctx, false)) {
			return
		}
		if diags.Extend(eng.LoadPluginRunner(ctx)) {
			return
		}
		res, diag := eng.FetchData(ctx, args[0])
		if diags.Extend(diag) {
			return
		}
		val := res.Any()
		ser, err := json.MarshalIndent(val, "", "    ")
		if diags.AppendErr(err, "Failed to serialize data output to json") {
			return
		}
		_, err = os.Stdout.Write(ser)

		diags.AppendErr(err, "Failed to output json data")
		return
	},
}
