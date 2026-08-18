// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package eval

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

type PluginDataAction struct {
	*PluginAction

	id string

	DataSource *plugin.DataSource
	SrcRange   hcl.Range
}

func (a *PluginDataAction) FetchData(ctx context.Context) (plugindata.Data, diagnostics.Diag) {
	res, diags := a.DataSource.Execute(ctx, &plugin.RetrieveDataParams{
		Config: a.Config,
		Args:   a.Args,
	})
	diags.Refine(diagnostics.DefaultSubject(a.SrcRange))
	return res, diags
}

func (a *PluginDataAction) ID() string {
	if a.id == "" {
		uid, err := uuid.NewV7()
		if err != nil {
			panic(fmt.Sprintf("Error generating a UUID v7: %v", err))
		}
		a.id = uid.String()
	}
	return a.id
}

// func (a *PluginDataAction) GetDetails() plugin.BlockDetails {
// 	return plugin.BlockDetails{
// 		ID:   a.id,
// 		Name: a.BlockName,
// 		RunnerName:    a.RunnerName,
// 		RunnerKind:    a.RunnerKind,
// 	}
// }

func LoadDataAction(
	ctx context.Context,
	sources DataSources,
	blockDef *definitions.DataBlock,
	dataCtx plugindata.Map,
) (_ *PluginDataAction, diags diagnostics.Diag) {
	defer func() {
		diags.Refine(diagnostics.DefaultSubject(blockDef.Source.Block.Range()))
	}()

	ds, ok := sources.DataSource(blockDef.RunnerName)
	if !ok {
		return nil, diagnostics.Diag{
			{
				Severity: hcl.DiagError,
				Summary:  "Missing data source",
				Detail: fmt.Sprintf(
					"`%s` data source not found in installed plugins",
					blockDef.RunnerName,
				),
			},
		}
	}
	var cfg *dataspec.Block
	if ds.Config != nil && blockDef.Config != nil {
		cfg, diags = blockDef.Config.ParseConfig(ctx, ds.Config)
		if diags.HasErrors() {
			return nil, diags
		}
	} else if ds.Config != nil && blockDef.Config == nil && ds.Config.IsRequired() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "No configuration provided for data block",
			Detail: fmt.Sprintf(
				"Data block for data source `%s` requires configuration but none was found.",
				blockDef.RunnerName,
			),
			Context: blockDef.Source.Block.Range().Ptr(),
		})
	} else if ds.Config == nil && blockDef.Config != nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Data source doesn't support configuration",
			Detail: fmt.Sprintf(
				"Data source `%s` doesn't support configuration, but was provided with one.",
				blockDef.RunnerName,
			),
			Subject: blockDef.Config.Range().Ptr(),
			Context: blockDef.Source.Block.Range().Ptr(),
		})
	}
	args, diag := dataspec.DecodeAndEvalBlock(ctx, blockDef.Source.Block, ds.Args, dataCtx)
	if diags.Extend(diag) {
		return nil, diags
	}
	return &PluginDataAction{
		PluginAction: &PluginAction{
			RunnerName: blockDef.RunnerName,
			BlockName:  blockDef.BlockName,
			meta:       blockDef.Meta,
			Config:     cfg,
			Args:       args,
			Source:     blockDef,
		},
		DataSource: ds,
	}, diags
}
