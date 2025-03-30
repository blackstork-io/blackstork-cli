package eval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/deferred"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

type PluginContentAction struct {
	*PluginAction
	Provider     *plugin.ContentProvider
	Vars         *definitions.ParsedVars
	RequiredVars []string
	DependsOn    []string
}

func (action *PluginContentAction) RenderContent(ctx context.Context, dataCtx plugindata.Map, doc, parent *plugin.ContentSection, contentID uint32) (diags diagnostics.Diag) {
	contentMap := plugindata.Map{}
	if action.PluginAction.Meta != nil {
		contentMap[definitions.BlockKindMeta] = action.PluginAction.Meta.AsPluginData()
	}

	// Create a clone of the data context to avoid modifying the original
	localDataCtx := dataCtx.Clone()

	docData := localDataCtx[definitions.BlockKindDocument]
	docData.(plugindata.Map)[definitions.BlockKindContent] = doc.AsData()
	localDataCtx[definitions.BlockKindDocument] = docData
	localDataCtx[definitions.BlockKindContent] = contentMap

	// Now apply the vars from the content block itself
	diag := ApplyVars(ctx, action.Vars, localDataCtx)
	if diags.Extend(diag) {
		return
	}

	isIncluded, diag := dataspec.EvalAttr(ctx, action.IsIncluded, localDataCtx)
	if diags.Extend(diag) {
		return
	}

	if isIncluded.IsNull() || !plugindata.IsTruthy(*plugindata.Encapsulated.MustFromCty(isIncluded)) {
		return
	}
	if len(action.RequiredVars) > 0 {
		diag = verifyRequiredVars(localDataCtx, action.RequiredVars, action.Source.Block)
		if diags.Extend(diag) {
			return
		}
	}

	evaluatedBlock, diag := dataspec.EvalBlockCopy(ctx, action.Args, localDataCtx)
	if diags.Extend(diag) {
		return
	}

	res, diag := action.Provider.Execute(ctx, &plugin.ProvideContentParams{
		Config:      action.Config,
		Args:        evaluatedBlock,
		DataContext: localDataCtx,
		ContentID:   contentID,
	})
	if diags.Extend(diag) {
		return
	}
	if res.Location == nil {
		res.Location = &plugin.Location{
			Index: contentID,
		}
	}
	if err := parent.Add(res.Content, res.Location); err != nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to add content",
			Detail:   fmt.Sprintf("Failed to add content: %s", err),
		})
		return
	}
	return
}

func LoadPluginContentAction(ctx context.Context, providers ContentProviders, node *definitions.ParsedPlugin) (_ *PluginContentAction, diags diagnostics.Diag) {
	cp, ok := providers.ContentProvider(node.PluginName)
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Missing content provider",
			Detail:   fmt.Sprintf("'%s' not found in any plugin", node.PluginName),
		}}
	}
	var cfg *dataspec.Block
	if cp.Config != nil {
		cfg, diags = node.Config.ParseConfig(ctx, cp.Config)
		if diags.HasErrors() {
			return nil, diags
		}
	} else if node.Config.Exists() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "ContentProvider doesn't support configuration",
			Detail: fmt.Sprintf("ContentProvider '%s' does not support configuration, "+
				"but was provided with one. Remove it.", node.PluginName),
			Subject: node.Config.Range().Ptr(),
			Context: node.Invocation.Range().Ptr(),
		})
		return nil, diags
	}

	args, diag := dataspec.DecodeBlock(
		deferred.WithQueryFuncs(ctx),
		node.Invocation.Block,
		cp.Args,
	)
	if diags.Extend(diag) {
		return nil, diags
	}
	isIncluded := node.IsIncluded
	if isIncluded == nil {
		isIncluded = defaultIsIncluded(node.Source.Block.DefRange())
	}

	isIncludedAttr, diag := dataspec.DecodeAttr(
		fabctx.GetEvalContext(deferred.WithQueryFuncs(ctx)),
		isIncluded,
		isIncludedSpec,
	)
	if diags.Extend(diag) {
		return nil, diags
	}
	return &PluginContentAction{
		PluginAction: &PluginAction{
			Source:     node.Source,
			PluginName: node.PluginName,
			BlockName:  node.BlockName,
			Meta:       node.Meta,
			Config:     cfg,
			Args:       args,
			IsIncluded: isIncludedAttr,
		},
		Provider:     cp,
		Vars:         node.Vars,
		RequiredVars: node.RequiredVars,
		DependsOn:    node.DependsOn,
	}, diags
}

var isIncludedSpec = &dataspec.AttrSpec{
	Name: "is_included",
	Type: plugindata.Encapsulated.CtyType(),
	Doc:  "Condition indicating whether content should be rendered",
}

func defaultIsIncluded(rng hcl.Range) *hclsyntax.Attribute {
	return &hclsyntax.Attribute{
		Name: definitions.AttrIsIncluded,
		Expr: &hclsyntax.LiteralValueExpr{
			Val:      plugindata.Encapsulated.ValToCty(plugindata.Bool(true)),
			SrcRange: rng,
		},
		SrcRange:    rng,
		NameRange:   rng,
		EqualsRange: rng,
	}
}
