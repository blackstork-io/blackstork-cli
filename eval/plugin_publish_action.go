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

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

type PluginPublishAction struct {
	*PluginAction
	Publisher *plugin.Publisher
	Format    *PluginFormatAction
}

func (block *PluginPublishAction) Publish(
	ctx context.Context,
	dataCtx plugindata.Map,
	content plugin.Content,
	formattedContent *plugin.FormattedContent,
	documentName string,
) diagnostics.Diag {
	return block.Publisher.Execute(ctx, &plugin.PublishParams{
		Config:           block.Config,
		Args:             block.Args,
		DataContext:      dataCtx,
		DocumentName:     documentName,
		Content:          content,
		FormattedContent: formattedContent,
	})
}

func LoadPluginPublishAction(
	ctx context.Context,
	publishers Publishers,
	formatters Formatters,
	blockDef *definitions.PublishBlock,
	dataCtx plugindata.Map,
) (_ *PluginPublishAction, diags diagnostics.Diag) {
	publisher, ok := publishers.Publisher(blockDef.RunnerName)
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Missing publisher",
			Detail:   fmt.Sprintf("'%s' not found in any plugin", blockDef.RunnerName),
		}}
	}
	var cfg *dataspec.Block
	if publisher.Config != nil && blockDef.Config != nil {
		cfg, diags = blockDef.Config.ParseConfig(ctx, publisher.Config)
		if diags.HasErrors() {
			return nil, diags
		}
	} else if publisher.Config != nil && blockDef.Config == nil && publisher.Config.IsRequired() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "No configuration provided for publish block",
			Detail: fmt.Sprintf(
				"Publish block for publisher `%s` requires configuration but none was found.",
				blockDef.RunnerName,
			),
			Context: blockDef.Source.Block.Range().Ptr(),
		})
	} else if publisher.Config == nil && blockDef.Config != nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Publisher doesn't support configuration",
			Detail: fmt.Sprintf(
				"Publisher '%s' doesn't support configuration, but one was provided.",
				blockDef.RunnerName,
			),
			Context: blockDef.Source.Block.Range().Ptr(),
		})
	}

	args, diag := dataspec.DecodeAndEvalBlock(ctx, blockDef.Source.Block, publisher.Args, dataCtx)
	if diags.Extend(diag) {
		return nil, diags
	}

	var publishFormat *PluginFormatAction

	if blockDef.Format != nil {
		format, diag := LoadPluginFormatAction(ctx, formatters, blockDef.Format, dataCtx)
		if diags.Extend(diag) {
			return nil, diags
		}
		publishFormat = format
	}

	return &PluginPublishAction{
		PluginAction: &PluginAction{
			RunnerName: blockDef.RunnerName,
			BlockName:  blockDef.BlockName,
			meta:       blockDef.Meta,
			Config:     cfg,
			Args:       args,
		},
		Publisher: publisher,
		Format:    publishFormat,
	}, diags
}
