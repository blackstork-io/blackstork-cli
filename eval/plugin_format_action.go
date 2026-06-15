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

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

type PluginFormatAction struct {
	*PluginAction
	Formatter *plugin.Formatter
}

func (block *PluginFormatAction) Execute(
	ctx context.Context,
	dataCtx plugindata.Map,
	content plugindata.Map,
) (*plugin.FormattedContent, diagnostics.Diag) {
	log := appctx.Log(ctx)
	log.InfoContext(
		ctx, "Formatting content",
		"format", block.Formatter.Format,
	)
	return block.Formatter.Execute(ctx, &plugin.FormatParams{
		Config:      block.Config,
		Args:        block.Args,
		Content:     content,
		DataContext: dataCtx,
		Format:      block.Formatter.Format,
	})
}

func LoadPluginFormatAction(
	ctx context.Context,
	formatters Formatters,
	blockDef *definitions.FormatBlock,
	dataCtx plugindata.Map,
) (_ *PluginFormatAction, diags diagnostics.Diag) {
	formatter, ok := formatters.Formatter(blockDef.RunnerName)
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Missing formatter",
			Detail:   fmt.Sprintf("'%s' not found in any plugin", blockDef.RunnerName),
		}}
	}
	var cfg *dataspec.Block
	if formatter.Config != nil && blockDef.Config != nil {
		cfg, diags = blockDef.Config.ParseConfig(ctx, formatter.Config)
		if diags.HasErrors() {
			return nil, diags
		}
	} else if formatter.Config != nil && blockDef.Config == nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "No configuration provided for format block",
			Detail: fmt.Sprintf(
				"Format block for formatter `%s` requires configuration but none was found.",
				blockDef.RunnerName,
			),
			Context: blockDef.Source.Block.Range().Ptr(),
		})
	} else if formatter.Config == nil && blockDef.Config != nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Formatter doesn't support configuration",
			Detail: fmt.Sprintf(
				"Formatter '%s' doesn't support configuration, but one was provided.",
				blockDef.RunnerName,
			),
			Context: blockDef.Source.Block.Range().Ptr(),
		})
	}

	args, diag := dataspec.DecodeAndEvalBlock(ctx, blockDef.Source.Block, formatter.Args, dataCtx)
	if diags.Extend(diag) {
		return nil, diags
	}

	return &PluginFormatAction{
		PluginAction: &PluginAction{
			RunnerName: blockDef.RunnerName,
			BlockName:  blockDef.BlockName,
			meta:       blockDef.Meta,
			Config:     cfg,
			Args:       args,
		},
		Formatter: formatter,
	}, diags
}

func LoadMarkdownPluginFormatAction(
	ctx context.Context,
	formatters Formatters,
) (_ *PluginFormatAction, diags diagnostics.Diag) {
	formatterName := "md"
	blockName := "default-md"

	p, ok := formatters.Formatter(formatterName)
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Missing formatter",
			Detail:   fmt.Sprintf("'%s' formatter not found in any plugin", formatterName),
		}}
	}

	return &PluginFormatAction{
		PluginAction: &PluginAction{
			RunnerName: formatterName,
			BlockName:  blockName,
		},
		Formatter: p,
	}, diags
}
