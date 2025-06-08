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

type PluginDataAction struct {
	*PluginAction
	DataSource   *plugin.DataSource
	SrcRange hcl.Range
}

func (action *PluginDataAction) FetchData(ctx context.Context) (plugindata.Data, diagnostics.Diag) {
	res, diags := action.DataSource.Execute(ctx, &plugin.RetrieveDataParams{
		Config: action.Config,
		Args:   action.Args,
	})
	diags.Refine(diagnostics.DefaultSubject(action.SrcRange))
	return res, diags
}

func LoadDataAction(
	ctx context.Context,
	sources DataSources,
	node *definitions.DataBlock,
) (_ *PluginDataAction, diags diagnostics.Diag) {
	defer func() {
		diags.Refine(diagnostics.DefaultSubject(node.Source.Block.Range()))
	}()

	ds, ok := sources.DataSource(node.BlockRunnerName)
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Missing data source",
			Detail:   fmt.Sprintf("`%s` data source not found in installed plugins", node.BlockRunnerName),
		}}
	}
	var cfgBlock *dataspec.Block
	if ds.Config != nil {
		cfgBlock, diags = node.Config.ParseConfig(ctx, ds.Config)
		if diags.HasErrors() {
			return nil, diags
		}
	} else if node.Config.Exists() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Data source doesn't support configuration",
			Detail: fmt.Sprintf(
				"Data source `%s` doesn't support configuration, but was provided with one.",
				node.BlockRunnerName),
			Subject: node.Config.Range().Ptr(),
			Context: node.Source.Block.Range().Ptr(),
		})
	}
	args, diag := dataspec.DecodeAndEvalBlock(ctx, node.Source.Block, ds.Args, nil)
	if diags.Extend(diag) {
		return nil, diags
	}
	return &PluginDataAction{
		PluginAction: &PluginAction{
			BlockRunnerName: node.BlockRunnerName,
			BlockName:  node.BlockName,
			meta:       node.Meta,
			Config:     cfgBlock,
			Args:       args,
			Source:     node,
		},
		DataSource: ds,
	}, diags
}
