package eval

import (
	"context"
	"reflect"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/parser"
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/deferred"
	"github.com/blackstork-io/fabric/plugin/plugindata"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

type Section struct {
	source *definitions.Section

	blockName string

	meta         *definitions.MetaBlock
	children     []ContentTreeEvalBlock
	vars         *definitions.Vars
	requiredVars []string

	dependsOn []EvalKey

	isIncluded *dataspec.Attr

	// To be filled in during unpacking and data propagation
	dataCtx          *plugindata.Map
	childrenToRender []RenderableContent
}

func (s *Section) EvalKey() EvalKey {
	return EvalKey{
		kind: definitions.BlockKindSection,
		name: s.blockName, // can be overriden by ref base or dynamic block
	}
}

func (s *Section) Kind() string {
	return s.source.GetSourceKind()
}

func (s *Section) GetDataCtx() *plugindata.Map {
	return s.dataCtx
}

func (s *Section) Meta() plugindata.Map {
	if s.meta == nil {
		return plugindata.Map{}
	}
	return s.meta.AsPluginData()
}

func (a *Section) addNameSuffix(val string) {
	a.blockName += ":" + val
}

func (s *Section) CtyType() cty.Type {
	nativeType := reflect.TypeOf(s)
	return cty.Capsule("section", nativeType)
}

func (block *Section) RenderContent(
	ctx context.Context,
	data plugindata.Map,
) (_ plugin.Content, diags diagnostics.Diag) {
	// Section's children are rendered async
	section := new(plugin.ContentSection)
	return section, nil
}

func (s *Section) GetDef() definitions.BlockDef {
	return s.source.Source
}

func (s *Section) isContentTreeEvalBlock() {}

var _ ContentTreeEvalBlock = (*Section)(nil)
var _ RenderableContent = (*Section)(nil)
var _ parser.Ctyable = (*Section)(nil)

func LoadSection(
	ctx context.Context,
	providers ContentProviders,
	sectionDef *definitions.Section,
) (_ *Section, diags diagnostics.Diag) {

	// Evaluate requiredVars lists
	var requiredVars []string
	for _, vars := range sectionDef.RequiredVarsCombined {
		var varNames []string
		diag := gohcl.DecodeExpression(vars.Expr, nil, &varNames)
		if diags.Extend(diag) {
			continue
		}
		requiredVars = append(requiredVars, varNames...)
	}

	// Evaluate dependsOn lists
	var dependsOn []EvalKey
	for _, deps := range sectionDef.DependsOnCombined {
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

	block := &Section{
		blockName:    sectionDef.BlockName,
		meta:         sectionDef.Meta,
		vars:         sectionDef.Vars,
		source:       sectionDef,
		requiredVars: requiredVars,
		dependsOn:    dependsOn,
	}
	var diag diagnostics.Diag
	isIncluded := sectionDef.IsIncluded
	if isIncluded == nil {
		isIncluded = defaultIsIncluded(sectionDef.Source.Block.DefRange())
	}

	block.isIncluded, diag = dataspec.DecodeAttr(
		fabctx.GetEvalContext(deferred.WithQueryFuncs(ctx)),
		isIncluded,
		isIncludedSpec,
	)
	if diags.Extend(diag) {
		return
	}

	if sectionDef.Title != nil {
		title, diag := LoadContent(ctx, providers, sectionDef.Title)
		if diags.Extend(diag) {
			return
		}
		block.children = append(block.children, title)

	}
	for _, child := range sectionDef.Content {
		decoded, diag := LoadContent(ctx, providers, child)
		if diags.Extend(diag) {
			return
		}
		block.children = append(block.children, decoded)
	}
	return block, diags
}
