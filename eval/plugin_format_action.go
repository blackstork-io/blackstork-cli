package eval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

type PluginFormatAction struct {
	*PluginAction
	Formatter *plugin.Formatter
}

func (block *PluginFormatAction) Execute(
	ctx context.Context,
	dataCtx plugindata.Map,
	content plugindata.Map,
	documentName string,
) (*plugin.FormattedContent, diagnostics.Diag) {
	return block.Formatter.Execute(ctx, &plugin.FormatParams{
		Config:      block.Config,
		Args:        block.Args,
		Content:     content,
		DataContext: dataCtx,
	})
}

func LoadPluginFormatAction(
	ctx context.Context,
	formatters Formatters,
	node *definitions.ParsedPlugin,
) (_ *PluginFormatAction, diags diagnostics.Diag) {
	p, ok := formatters.Formatter(node.PluginName)
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Missing formatter",
			Detail:   fmt.Sprintf("'%s' not found in any plugin", node.PluginName),
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
			Summary:  "Formatter doesn't support configuration",
			Detail: fmt.Sprintf(
				"Formatter '%s' does not support configuration, but was provided with one",
				node.PluginName),
			Subject: node.Config.Range().Ptr(),
			Context: node.Invocation.Range().Ptr(),
		})
		return nil, diags
	}

	// 	var format string
	// 	if attr, found := utils.Pop(node.Invocation.Body.Attributes, "format"); found {
	// 		val, diag := dataspec.DecodeAttr(fabctx.GetEvalContext(ctx), attr, &dataspec.AttrSpec{
	// 			Name:        "format",
	// 			Type:        cty.String,
	// 			Constraints: constraint.RequiredMeaningful,
	// 			OneOf: constraint.OneOf(
	// 				utils.FnMap(p.Format, func(f string) cty.Value {
	// 					return cty.StringVal(f)
	// 				})),
	// 		})
	//
	// 		if diags.Extend(diag) {
	// 			return
	// 		}
	// 		format = val.Value.AsString()
	// 	}

	args, diag := dataspec.DecodeAndEvalBlock(ctx, node.Invocation.Block, p.Args, nil)
	if diags.Extend(diag) {
		return nil, diags
	}
	return &PluginFormatAction{
		PluginAction: &PluginAction{
			PluginName: node.PluginName,
			BlockName:  node.BlockName,
			Meta:       node.Meta,
			Config:     cfg,
			Args:       args,
		},
		Formatter: p,
	}, diags
}

func LoadMarkdownPluginFormatAction(
	ctx context.Context,
	formatters Formatters,
) (_ *PluginFormatAction, diags diagnostics.Diag) {
	pluginName := "md"

	p, ok := formatters.Formatter(pluginName)
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Missing formatter",
			Detail:   fmt.Sprintf("'%s' not found in any plugin", pluginName),
		}}
	}

	return &PluginFormatAction{
		PluginAction: &PluginAction{
			PluginName: pluginName,
		},
		Formatter: p,
	}, diags
}
