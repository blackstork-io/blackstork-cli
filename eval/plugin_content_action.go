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
	"reflect"

	"github.com/google/uuid"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/deferred"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

type PluginContentAction struct {
	PluginAction

	Provider     *plugin.ContentProvider
	vars         *definitions.Vars
	requiredVars []string
	dependsOn    []EvalKey

	isIncluded *dataspec.Attr

	id string

	// To be filled in during unpacking and data propagation
	dataCtx *plugindata.Map
}

func (a *PluginContentAction) EvalKey() EvalKey {
	defKey := a.Source.GetSource().Key()
	return EvalKey{
		Kind:   defKey.Kind,
		Runner: a.RunnerName, // can be overriden by ref base
		Name:   a.BlockName,  // can be overriden by ref base or dynamic block
	}
}

func (a *PluginContentAction) Kind() string {
	return a.Source.GetSource().Kind()
}

func (a *PluginContentAction) GetDataCtx() *plugindata.Map {
	return a.dataCtx
}

func (a *PluginContentAction) Clone(suffix string) ContentTreeEvalBlock {
	c := *a
	r := &c
	r.makeNewID()
	if suffix != "" {
		r.addNameSuffix(suffix)
	}
	return r
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

func (a *PluginContentAction) makeNewID() {
	uid, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Sprintf("Error generating a UUID v7: %v", err))
	}
	a.id = uid.String()
}

func (a *PluginContentAction) ID() string {
	if a.id == "" {
		a.makeNewID()
	}
	return a.id
}

func (a *PluginContentAction) RenderContent(
	ctx context.Context,
	dataCtx plugindata.Map,
) (_ plugin.Content, diags diagnostics.Diag) {
	evaluatedBlock, diag := dataspec.EvalBlockCopy(ctx, a.Args, dataCtx)
	if diags.Extend(diag) {
		return
	}

	res, err := a.Provider.Execute(ctx, &plugin.ProvideContentParams{
		Config:      a.Config,
		Args:        evaluatedBlock,
		DataContext: dataCtx,
	})
	if err != nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary: fmt.Sprintf(
				"Error while rendering content for %s: %s",
				a.EvalKey().AsName(),
				err,
			),
			Detail: err.Error(),
		})
		return nil, diags
	}
	return res.Content, nil
}

func (a *PluginContentAction) GetDef() definitions.BlockDef {
	return a.Source.GetSource()
}

func (a *PluginContentAction) isContentTreeEvalBlock() {}

var (
	_ ContentTreeEvalBlock = (*PluginContentAction)(nil)
	_ RenderableContent    = (*PluginContentAction)(nil)
)

// var _ parser.Ctyable = (*PluginContentAction)(nil)

func LoadPluginContentAction(
	ctx context.Context,
	providers ContentProviders,
	blockDef *definitions.ContentBlock,
) (_ *PluginContentAction, diags diagnostics.Diag) {
	provider, ok := providers.ContentProvider(blockDef.RunnerName)
	if !ok {
		return nil, diagnostics.Diag{
			{
				Severity: hcl.DiagError,
				Summary:  "Content provider is not found",
				Detail: fmt.Sprintf(
					"'%s' content provider is not found in installed plugins",
					blockDef.RunnerName,
				),
			},
		}
	}
	var cfg *dataspec.Block
	if provider.Config != nil && blockDef.Config != nil {
		cfg, diags = blockDef.Config.ParseConfig(ctx, provider.Config)
		if diags.HasErrors() {
			return nil, diags
		}
	} else if provider.Config != nil && blockDef.Config == nil && provider.Config.IsRequired() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "No configuration provided for content block",
			Detail: fmt.Sprintf(
				"Content block for content provider `%s` requires configuration but none was found.",
				blockDef.RunnerName,
			),
			Context: blockDef.Source.Block.Range().Ptr(),
		})
	} else if provider.Config == nil && blockDef.Config != nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Content provider doesn't support configuration",
			Detail: fmt.Sprintf(
				"Content provider '%s' doesn't support configuration, but was provided with one.",
				blockDef.RunnerName,
			),
			Context: blockDef.Source.Block.Range().Ptr(),
		})
	}

	evalCtx := deferred.WithQueryFuncs(appctx.GetEvalContext(ctx))

	args, diag := dataspec.DecodeBlock(
		evalCtx,
		blockDef.Source.Block,
		provider.Args,
	)
	if diags.Extend(diag) {
		return nil, diags
	}
	isIncluded := blockDef.IsIncluded
	if isIncluded == nil {
		isIncluded = defaultIsIncluded(blockDef.Source.Block.DefRange())
	}

	isIncludedAttr, diag := dataspec.DecodeAttr(
		evalCtx,
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
			RunnerName: blockDef.RunnerName,
			BlockName:  blockDef.BlockName,
			meta:       blockDef.Meta,
			Config:     cfg,
			Args:       args,
			Source:     blockDef,
		},
		Provider:     provider,
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
