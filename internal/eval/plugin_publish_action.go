package eval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/utils"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
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
		FormattedContent: formattedContent,
	})
}

func LoadPluginPublishAction(
	ctx context.Context,
	publishers Publishers,
	node *definitions.PublishBlock,
) (_ *PluginPublishAction, diags diagnostics.Diag) {
	p, ok := publishers.Publisher(node.BlockRunnerName)
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Missing publisher",
			Detail:   fmt.Sprintf("'%s' not found in any plugin", node.BlockRunnerName),
		}}
	}
	var cfg *dataspec.Block
	if p.Config != nil {
		cfg, diags = node.Config.ParseConfig(ctx, p.Config)
		if diags.HasErrors() {
			return nil, diags
		}
	} else if node.Config.Exists() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Publisher doesn't support configuration",
			Detail: fmt.Sprintf(
				"Publisher `%s` doesn't support configuration, but was provided with one.",
				node.BlockRunnerName),
			Subject: node.Config.Range().Ptr(),
			Context: node.Source.Block.Range().Ptr(),
		})
		return nil, diags
	}

	var format *string
	formatAttr, found := utils.Pop(node.Source.Block.Body.Attributes, "format")

	if found && len(p.Formats) > 0 {
		val, diag := dataspec.DecodeAttr(fabctx.GetEvalContext(ctx), formatAttr, &dataspec.AttrSpec{
			Name:        "format",
			Type:        cty.String,
			Constraints: constraint.RequiredMeaningful,
			// FIXME: how does it work with an empty Formats list?
			OneOf: constraint.OneOf(
				utils.FnMap(p.Formats, func(f string) cty.Value {
					return cty.StringVal(f)
				})),
		})
		if diags.Extend(diag) {
			return
		}
		formatStr := val.Value.AsString()
		format = &formatStr
	} else if found && len(p.Formats) == 0 {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Publisher doesn't support format specification",
			Detail: fmt.Sprintf(
				"Publisher '%s' does not support format specification, but was provided with one.",
				node.BlockRunnerName,
			),
			Subject: node.Config.Range().Ptr(),
			Context: node.Source.Block.Range().Ptr(),
		})
		return nil, diags
	} else if !found && len(p.Formats) > 0 {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "No format specified for publisher",
			Detail: fmt.Sprintf("Format value must be set for the publisher '%s'", node.BlockRunnerName),
			Subject: node.Config.Range().Ptr(),
			Context: node.Source.Block.Range().Ptr(),
		})
		return nil, diags
	}

	args, diag := dataspec.DecodeAndEvalBlock(ctx, node.Source.Block, p.Args, nil)
	if diags.Extend(diag) {
		return nil, diags
	}
	return &PluginPublishAction{
		PluginAction: &PluginAction{
			BlockRunnerName: node.BlockRunnerName,
			BlockName:  node.BlockName,
			meta:       node.Meta,
			Config:     cfg,
			Args:       args,
		},
		Publisher: p,
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
			BlockRunnerName: publisherName,
			BlockName: blockName,
		},
		Publisher: publisher,
		Format:    &format,
	}, nil
}

