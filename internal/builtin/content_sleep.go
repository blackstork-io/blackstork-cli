package builtin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/constraint"
)

func makeSleepContentProvider(log *slog.Logger) *plugin.ContentProvider {
	return &plugin.ContentProvider{
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
		ContentFunc: func(ctx context.Context, params *plugin.ProvideContentParams) (*plugin.ContentProviderResult, diagnostics.Diag) {
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

			log.WarnContext(ctx, "Sleeping", "duration", duration)
			time.Sleep(duration)

			return &plugin.ContentProviderResult{
				Content: plugin.NewTextElement(fmt.Sprintf("Slept for %s.", duration)),
			}, nil
		},
	}
}
