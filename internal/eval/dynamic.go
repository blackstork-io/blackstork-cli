package eval

import (
	"context"
	"fmt"
	"maps"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/deferred"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

const (
	itemIndexVarName = "dynamic_item_index"
	itemKeyVarName   = "dynamic_item_key"
	itemValueVarName = "dynamic_item"
)

type Dynamic struct {
	blockName string
	source    *definitions.Dynamic

	items *dataspec.Attr

	children []ContentTreeEvalBlock

	dependsOn []EvalKey
}

func (d *Dynamic) isContentTreeEvalBlock() {}

func (d *Dynamic) EvalKey() EvalKey {
	return EvalKey{
		kind: d.Kind(),
		name: d.blockName,
	}
}

func (d *Dynamic) Kind() string {
	return definitions.BlockKindDynamic
}

func (d *Dynamic) addNameSuffix(val string) {
	d.blockName += ":" + val
}

var dynamicBlockItems = &dataspec.AttrSpec{
	Name:        "items",
	Type:        plugindata.Encapsulated.CtyType(),
	Doc:         "A list or a map of items to be iterated over",
	Constraints: constraint.Required,
}

func LoadDynamic(
	ctx context.Context,
	providers ContentProviders,
	dynamicDef *definitions.Dynamic,
) (_ *Dynamic, diags diagnostics.Diag) {
	var diag diagnostics.Diag
	block := &Dynamic{
		source:    dynamicDef,
		blockName: dynamicDef.BlockName,
		// block:    dynamicDef.Source.Block,
		children: make([]ContentTreeEvalBlock, 0, len(dynamicDef.Children)),
	}
	evalCtx := fabctx.GetEvalContext(deferred.WithQueryFuncs(ctx))
	block.items, diag = dataspec.DecodeAttr(evalCtx, dynamicDef.Items, dynamicBlockItems)
	diags.Extend(diag)

	// Evaluate dependsOn lists
	if dynamicDef.DependsOn != nil {
		var depNames []string
		diag := gohcl.DecodeExpression(dynamicDef.DependsOn.Expr, nil, &depNames)
		if diags.Extend(diag) {
			return nil, diags
		}
		for _, name := range depNames {
			defKey, err := definitions.KeyFromName(name)
			if err != nil {
				diags.Extend(diagnostics.FromErr(err))
				return nil, diags
			}
			evalKey := EvalKeyFromDefKey(*defKey)
			block.dependsOn = append(block.dependsOn, evalKey)
		}
	}

	for _, child := range dynamicDef.Children {
		decoded, diag := LoadContent(ctx, providers, child)
		diags.Extend(diag)
		block.children = append(block.children, decoded)
	}
	return block, diags
}

func evalItemsAttr(
	ctx context.Context,
	items *dataspec.Attr,
	dataCtx *plugindata.Map,
) (data *plugindata.Data, diags diagnostics.Diag) {
	val, diag := dataspec.EvalAttr(ctx, items, *dataCtx)
	if diags.Extend(diag) || val.IsNull() {
		return nil, diags
	}
	data = plugindata.Encapsulated.MustFromCty(val)
	if data == nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid `items` value for a dynamic block",
			Detail:   "The value of `items` attribute for a dynamic block must be a list or a map.",
			Subject:  items.ValueRange.Ptr(),
		})
		return nil, diags
	}
	return data, diags
}

type dynamicItem struct {
	idx int
	key *string
	val plugindata.Data
}

func evaluateDynamicBlock(
	ctx context.Context,
	requiredTags []string,
	dynamic *Dynamic,
	items plugindata.Data,
	dataCtx *plugindata.Map,
) (res []RenderableContent, diags diagnostics.Diag) {

	var dynamicItems []dynamicItem
	switch items := (items).(type) {
	case nil:
		return
	case plugindata.List:
		dynamicItems = make([]dynamicItem, 0, len(items))
		for idx, item := range items {
			dynamicItems = append(dynamicItems, dynamicItem{
				idx: idx,
				val: item,
			})
		}
	case plugindata.Map:
		dynamicItems = make([]dynamicItem, 0, len(items))
		idx := 0
		for key, item := range items {
			dynamicItems = append(dynamicItems, dynamicItem{
				idx: idx,
				key: &key,
				val: item,
			})
			idx += 1
		}
	default:
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid dynamic block items type",
			Detail:   fmt.Sprintf("Dynamic block items value must be a list or a map, got %T", items),
			Subject:  dynamic.items.ValueRange.Ptr(),
		})
		return
	}

	for _, item := range dynamicItems {
		// clone the context for the run
		newDataCtx := maps.Clone(*dataCtx)

		// add dynamic vars to the context
		var vars plugindata.Map
		varsData, found := newDataCtx[definitions.BlockKindVars]
		if found {
			vars = varsData.(plugindata.Map)
		} else {
			vars = plugindata.Map{}
		}
		vars[itemIndexVarName] = plugindata.Number(item.idx)
		vars[itemValueVarName] = item.val

		if item.key != nil {
			vars[itemKeyVarName] = plugindata.String(*item.key)
		}
		newDataCtx[definitions.BlockKindVars] = vars

		// clone the children
		for _, child := range dynamic.children {
			var childClone ContentTreeEvalBlock

			switch childVal := child.(type) {
			case *PluginContentAction:
				clone := *childVal
				childClone = &clone
			case *Section:
				clone := *childVal
				childClone = &clone
			case *Dynamic:
				clone := *childVal
				childClone = &clone
			default:
				diags.Add(
					"Unknown type of a content tree block",
					fmt.Sprintf(
						"Block `%s` type `%T` encountered while evaluating a content tree",
						childVal.EvalKey().AsName(),
						childVal,
					),
				)
				return nil, diags
			}

			childClone.addNameSuffix(fmt.Sprintf("%d", item.idx))

			nonDynamicContent, diag := evaluateContentTree(ctx, requiredTags, childClone, &newDataCtx)
			if diags.Extend(diag) {
				// stop dynamic block processing on error: it's likely that
				// the error will be repeated for each item and only add noise
				break
			}
			if nonDynamicContent != nil {
				res = append(res, nonDynamicContent)
			}
		}
	}
	return
}

// func parseDynVars(
// 	ctx context.Context,
// 	idx, val plugindata.Data,
// 	rng hcl.Range,
// ) (parsed *definitions.Vars, diags diagnostics.Diag) {
// 	// use existing vars parser by creating a synthetic (dynamic_)vars block
// 	return parser.ParseVars(ctx, &hclsyntax.Block{
// 		Type:            "dynamic_vars",
// 		TypeRange:       rng,
// 		OpenBraceRange:  rng,
// 		CloseBraceRange: rng,
// 		Body: &hclsyntax.Body{
// 			SrcRange: rng,
// 			EndRange: rng,
// 			Attributes: map[string]*hclsyntax.Attribute{
// 				itemIndexVarName: {
// 					Name: itemIndexVarName,
// 					Expr: &hclsyntax.LiteralValueExpr{
// 						Val: plugindata.Encapsulated.ValToCty(idx),
// 					},
// 					SrcRange:    rng,
// 					NameRange:   rng,
// 					EqualsRange: rng,
// 				},
// 				itemVarName: {
// 					Name: itemVarName,
// 					Expr: &hclsyntax.LiteralValueExpr{
// 						Val: plugindata.Encapsulated.ValToCty(val),
// 					},
// 					SrcRange:    rng,
// 					NameRange:   rng,
// 					EqualsRange: rng,
// 				},
// 			},
// 		},
// 	}, nil)
// }
