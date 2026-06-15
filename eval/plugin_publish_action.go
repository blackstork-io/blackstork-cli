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
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

type PluginPublishAction struct {
	*PluginAction
	Publisher *plugin.Publisher
	Format    *string
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
	} else if publisher.Config != nil && blockDef.Config == nil {
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

	var format *string
	formatAttr, found := utils.Pop(blockDef.Source.Block.Body.Attributes, "format")

	if found && len(publisher.Formats) > 0 {
		val, diag := dataspec.DecodeAttr(appctx.GetEvalContext(ctx), formatAttr, &dataspec.AttrSpec{
			Name:        "format",
			Type:        cty.String,
			Constraints: constraint.RequiredMeaningful,
			// FIXME: how does it work with an empty Formats list?
			OneOf: constraint.OneOf(
				utils.FnMap(publisher.Formats, func(f string) cty.Value {
					return cty.StringVal(f)
				}),
			),
		})
		if diags.Extend(diag) {
			return nil, diags
		}
		formatStr := val.Value.AsString()
		format = &formatStr
	} else if found && len(publisher.Formats) == 0 {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Publisher doesn't support format specification",
			Detail: fmt.Sprintf(
				"Publisher '%s' does not support format specification, but was provided with one.",
				blockDef.RunnerName,
			),
			Subject: blockDef.Config.Range().Ptr(),
			Context: blockDef.Source.Block.Range().Ptr(),
		})
	} else if !found && len(publisher.Formats) > 0 {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "No format specified for publisher",
			Detail:   fmt.Sprintf("Format value must be set for the publisher '%s'", blockDef.RunnerName),
			Subject:  blockDef.Config.Range().Ptr(),
			Context:  blockDef.Source.Block.Range().Ptr(),
		})
		return nil, diags
	}

	args, diag := dataspec.DecodeAndEvalBlock(ctx, blockDef.Source.Block, publisher.Args, dataCtx)
	if diags.Extend(diag) {
		return nil, diags
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
		Format:    format,
	}, diags
}

func LoadStdoutPluginPublishAction(ctx context.Context, publishers Publishers, format string) (_ *PluginPublishAction, diags diagnostics.Diag) {
	publisherName := "stdout"
	blockName := "default-stdout"

	publisher, ok := publishers.Publisher(publisherName)
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Missing publisher",
			Detail:   fmt.Sprintf("Publisher `%s` not found in the installed plugins", publisherName),
		}}
	}

	return &PluginPublishAction{
		PluginAction: &PluginAction{
			RunnerName: publisherName,
			BlockName:  blockName,
		},
		Publisher: publisher,
		Format:    &format,
	}, nil
}
