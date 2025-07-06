package builtin

import (
	"context"
	"log/slog"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

func makeSleepDataSource(logger *slog.Logger) *plugin.DataSource {
	logger = logger.With("data_source", "sleep")

	return &plugin.DataSource{
		Doc: `
			Sleeps for the specified duration. Useful for testing and debugging.
		`,
		Tags: []string{"debug"},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "duration",
					Type:        cty.String,
					Doc:         "Duration to sleep",
					Constraints: constraint.Meaningful,
					DefaultVal:  cty.StringVal("1s"),
				},
			},
		},
		DataFunc: func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
			duration, err := time.ParseDuration(params.Args.GetAttrVal("duration").AsString())
			if err != nil {
				return nil, diagnostics.Diag{
					{
						Severity: hcl.DiagError,
						Summary:  "Invalid duration",
						Detail:   err.Error(),
					},
				}
			}

			logger.WarnContext(ctx, "Sleeping", "duration", duration)

			startTime := time.Now()
			time.Sleep(duration)
			endTime := time.Now()

			return plugindata.Map{
				"start_time": plugindata.String(startTime.Format(time.RFC3339)),
				"took":       plugindata.String(duration.String()),
				"end_time":   plugindata.String(endTime.Format(time.RFC3339)),
			}, nil
		},
	}
}
