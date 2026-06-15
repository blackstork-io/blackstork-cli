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

	"github.com/google/uuid"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/deferred"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
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

	id string
}

func (d *Dynamic) isContentTreeEvalBlock() {}

func (d *Dynamic) EvalKey() EvalKey {
	return EvalKey{
		Kind: d.Kind(),
		Name: d.blockName,
	}
}

func (d *Dynamic) Kind() string {
	return definitions.BlockKindDynamic
}

func (d *Dynamic) addNameSuffix(val string) {
	d.blockName += ":" + val
}

func (d *Dynamic) makeNewID() {
	uid, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Sprintf("Error generating a UUID v7: %v", err))
	}
	d.id = uid.String()
}

func (d *Dynamic) ID() string {
	if d.id == "" {
		d.makeNewID()
	}
	return d.id
}

func (d *Dynamic) Clone(suffix string) ContentTreeEvalBlock {
	res := &Dynamic{
		blockName: d.blockName,
		source:    d.source,
		items:     d.items,
		children:  utils.FnMap(d.children, func(b ContentTreeEvalBlock) ContentTreeEvalBlock { return b.Clone(suffix) }),
		dependsOn: d.dependsOn,
	}
	res.makeNewID()
	if suffix != "" {
		res.addNameSuffix(suffix)
	}
	return res
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
	evalCtx := appctx.GetEvalContext(deferred.WithQueryFuncs(ctx))
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
	depth int,
	dataCtx *plugindata.Map,
) (_ []RenderableContent, diags diagnostics.Diag) {
	var dynamicItems []dynamicItem
	switch items := items.(type) {
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

	res := []RenderableContent{}

	for _, item := range dynamicItems {
		// clone the context for the run
		newDataCtx := dataCtx.Clone()

		// add dynamic vars to the context
		vars := plugindata.Map{}
		if varsData, found := newDataCtx[definitions.BlockKindVars]; found {
			vars = varsData.(plugindata.Map)
		}

		vars[itemIndexVarName] = plugindata.Number(item.idx)
		vars[itemValueVarName] = item.val

		if item.key != nil {
			vars[itemKeyVarName] = plugindata.String(*item.key)
		}
		newDataCtx[definitions.BlockKindVars] = vars

		// clone the children
		for _, child := range dynamic.children {
			clone := child.Clone(fmt.Sprintf("%d", item.idx))

			nonDynamicContent, diag := evaluateContentTree(ctx, requiredTags, clone, depth, &newDataCtx)
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
	return res, diags
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
