package eval

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/parser"
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/deferred"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

type PluginContentAction struct {
	PluginAction

	Provider     *plugin.ContentProvider
	vars         *definitions.Vars
	requiredVars []string
	dependsOn    []EvalKey

	isIncluded *dataspec.Attr

	// To be filled in during unpacking and data propagation
	dataCtx *plugindata.Map
}

func (a *PluginContentAction) EvalKey() EvalKey {
	defKey := a.Source.GetSource().Key()
	return EvalKey{
		kind:   defKey.Kind,
		runner: a.BlockRunnerName, // can be overriden by ref base
		name:   a.BlockName,       // can be overriden by ref base or dynamic block
	}
}

func (a *PluginContentAction) Kind() string {
	return a.Source.GetSource().Kind()
}

func (a *PluginContentAction) GetDataCtx() *plugindata.Map {
	return a.dataCtx
}

func (a *PluginContentAction) Meta() plugindata.Map {
	if a.meta == nil {
		return plugindata.Map{}
	}
	return a.meta.AsPluginData()
}

func (a *PluginContentAction) CtyType() cty.Type {
	nativeType := reflect.TypeOf(a)
	return cty.Capsule(a.Kind(), nativeType)
}

func (a *PluginContentAction) addNameSuffix(val string) {
	a.BlockName += ":" + val
}

func (action *PluginContentAction) RenderContent(
	ctx context.Context,
	dataCtx plugindata.Map,
) (_ plugin.Content, diags diagnostics.Diag) {
	evaluatedBlock, diag := dataspec.EvalBlockCopy(ctx, action.Args, dataCtx)
	if diags.Extend(diag) {
		return
	}

	res, diag := action.Provider.Execute(ctx, &plugin.ProvideContentParams{
		Config:      action.Config,
		Args:        evaluatedBlock,
		DataContext: dataCtx,
	})
	if diags.Extend(diag) {
		return
	}
	return res.Content, nil
}

func (s *PluginContentAction) GetDef() definitions.BlockDef {
	return s.Source.GetSource()
}

func (s *PluginContentAction) isContentTreeEvalBlock() {}

var _ ContentTreeEvalBlock = (*PluginContentAction)(nil)
var _ RenderableContent = (*PluginContentAction)(nil)
var _ parser.Ctyable = (*PluginContentAction)(nil)

func LoadPluginContentAction(
	ctx context.Context,
	providers ContentProviders,
	blockDef *definitions.ContentBlock,
) (_ *PluginContentAction, diags diagnostics.Diag) {
	cp, ok := providers.ContentProvider(blockDef.BlockRunnerName)
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Content provider is not found",
			Detail:   fmt.Sprintf("'%s' content provider is not found in installed plugins", blockDef.BlockRunnerName),
		}}
	}
	var cfg *dataspec.Block
	if cp.Config != nil {
		cfg, diags = blockDef.Config.ParseConfig(ctx, cp.Config)
		if diags.HasErrors() {
			return nil, diags
		}
	} else if blockDef.Config.Exists() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Content provider doesn't support configuration",
			Detail: fmt.Sprintf(
				"Content provider '%s' doesn't support configuration, but was provided with one.",
				blockDef.BlockRunnerName),
			Subject: blockDef.Config.Range().Ptr(),
			Context: blockDef.Source.Block.Range().Ptr(),
		})
		return nil, diags
	}

	args, diag := dataspec.DecodeBlock(
		deferred.WithQueryFuncs(ctx),
		blockDef.Source.Block,
		cp.Args,
	)
	if diags.Extend(diag) {
		return nil, diags
	}
	isIncluded := blockDef.IsIncluded
	if isIncluded == nil {
		isIncluded = defaultIsIncluded(blockDef.Source.Block.DefRange())
	}

	isIncludedAttr, diag := dataspec.DecodeAttr(
		fabctx.GetEvalContext(deferred.WithQueryFuncs(ctx)),
		isIncluded,
		isIncludedSpec,
	)
	if diags.Extend(diag) {
		return nil, diags
	}

	// Evaluate requiredVars lists
	var requiredVars []string
	for _, vars := range blockDef.RequiredVarsCombined {
		var varNames []string
		diag := gohcl.DecodeExpression(vars.Expr, nil, &varNames)
		if diags.Extend(diag) {
			continue
		}
		requiredVars = append(requiredVars, varNames...)
	}

	// Evaluate dependsOn lists
	var dependsOn []EvalKey
	for _, deps := range blockDef.DependsOnCombined {
		var depNames []string
		diag := gohcl.DecodeExpression(deps.Expr, nil, &depNames)
		if diags.Extend(diag) {
			continue
		}

		for _, name := range depNames {
			defKey, err := definitions.KeyFromName(name)
			if err != nil {
				diags.Extend(diagnostics.FromErr(err))
				return nil, diags
			}
			evalKey := EvalKeyFromDefKey(*defKey)
			dependsOn = append(dependsOn, evalKey)
		}
	}

	return &PluginContentAction{
		PluginAction: PluginAction{
			BlockRunnerName: blockDef.BlockRunnerName,
			BlockName:       blockDef.BlockName,
			meta:            blockDef.Meta,
			Config:          cfg,
			Args:            args,
			Source:          blockDef,
		},
		Provider:     cp,
		vars:         blockDef.Vars,
		requiredVars: requiredVars,
		dependsOn:    dependsOn,
		isIncluded:   isIncludedAttr,
	}, diags
}

var isIncludedSpec = &dataspec.AttrSpec{
	Name: definitions.AttrIsIncluded,
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
