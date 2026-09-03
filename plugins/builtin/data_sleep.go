// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package builtin

import (
	"context"
	"log/slog"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeSleepDataSource(log *slog.Logger) *plugin.DataSource {
	log = log.With("data_source", "sleep")

	return &plugin.DataSource{
		Doc:  "Pauses data retrieval for the specified duration and returns the start time, end time, and elapsed duration. Intended for testing and debugging.",
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

			log.WarnContext(ctx, "Sleeping", "duration", duration)

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
