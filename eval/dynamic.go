package eval

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/parser"
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/utils"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/plugin/dataspec/deferred"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

type Dynamic struct {
	block    *hclsyntax.Block
	items    *dataspec.Attr
	children []*Content
}

var dynamicBlockItems = &dataspec.AttrSpec{
	Name:        "items",
	Type:        plugindata.Encapsulated.CtyType(),
	Doc:         "Items to be iterated over (list or map)",
	Constraints: constraint.Required,
}

func LoadDynamic(ctx context.Context, providers ContentProviders, node *definitions.ParsedDynamic) (_ *Dynamic, diags diagnostics.Diag) {
	var diag diagnostics.Diag
	block := &Dynamic{
		block:    node.Block,
		children: make([]*Content, 0, len(node.Content)),
	}
	evalCtx := fabctx.GetEvalContext(deferred.WithQueryFuncs(ctx))
	block.items, diag = dataspec.DecodeAttr(evalCtx, node.Items, dynamicBlockItems)
	diags.Extend(diag)

	for _, child := range node.Content {
		decoded, diag := LoadContent(ctx, providers, child)
		diags.Extend(diag)
		block.children = append(block.children, decoded)
	}
	return block, diags
}

func applyDynamicContentVars(ctx context.Context, children []*Content, dataCtx plugindata.Map, dynVarVals *definitions.ParsedVars) (res []*Content, diags diagnostics.Diag) {
	res = make([]*Content, 0, len(children))

	// unwrap dynamic content
	for _, child := range children {
		switch {
		case child.Plugin != nil:
			plugin := utils.Clone(child.Plugin)
			plugin.Vars = plugin.Vars.MergeWithBaseVars(dynVarVals)
			res = append(res, &Content{Plugin: plugin})
		case child.Section != nil:
			section := utils.Clone(child.Section)
			section.vars = section.vars.MergeWithBaseVars(dynVarVals)
			res = append(res, &Content{Section: section})
		case child.Dynamic != nil:
			nonDynamicContent, diag := unwrapDynamicItem(ctx, child.Dynamic, dataCtx)
			diags.Extend(diag)
			res = append(res, nonDynamicContent...)
		default:
			slog.Warn("Child has unknown type")
			res = append(res, child)
		}
	}
	return
}

// unwrapDynamicContent unwraps dynamic content in children
func UnwrapDynamicContent(ctx context.Context, children []*Content, dataCtx plugindata.Map) (res []*Content, diags diagnostics.Diag) {
	return unwrapDynamicContent(deferred.WithQueryFuncs(ctx), children, dataCtx)
}

func unwrapDynamicContent(ctx context.Context, children []*Content, dataCtx plugindata.Map) (res []*Content, diags diagnostics.Diag) {
	// Goal: expand dynamic content on the first layer of children
	// (without descending into child sections)
	res = make([]*Content, 0, len(children))

	// unwrap dynamic content
	for _, child := range children {
		if child.Dynamic == nil {
			res = append(res, child)
			continue
		}
		nonDynamicContent, diag := unwrapDynamicItem(ctx, child.Dynamic, dataCtx)
		diags.Extend(diag)
		res = append(res, nonDynamicContent...)
	}
	return
}

func unwrapDynamicItem(ctx context.Context, dynamic *Dynamic, dataCtx plugindata.Map) (res []*Content, diags diagnostics.Diag) {
	val, diag := dataspec.EvalAttr(ctx, dynamic.items, dataCtx)
	if diags.Extend(diag) || val.IsNull() {
		return
	}
	data := plugindata.Encapsulated.MustFromCty(val)
	if data == nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid dynamic block items",
			Detail:   "Dynamic block items must be a list or a map, got nil",
			Subject:  dynamic.items.ValueRange.Ptr(),
		})
		return
	}

	var dynamicItems [][2]plugindata.Data
	switch dt := (*data).(type) {
	case nil:
		return
	case plugindata.List:
		dynamicItems = make([][2]plugindata.Data, 0, len(dt))
		for idx, item := range dt {
			dynamicItems = append(dynamicItems, [2]plugindata.Data{
				plugindata.Number(idx),
				item,
			})
		}
	case plugindata.Map:
		dynamicItems = make([][2]plugindata.Data, 0, len(dt))
		for key, item := range dt {
			dynamicItems = append(dynamicItems, [2]plugindata.Data{
				plugindata.String(key),
				item,
			})
		}
	default:
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid dynamic block items",
			Detail:   fmt.Sprintf("Dynamic block items must be a list or a map, got %T", dt),
			Subject:  dynamic.items.ValueRange.Ptr(),
		})
		return
	}

	newDataCtx := maps.Clone(dataCtx)
	vars := getVarsCopy(newDataCtx)
	for _, kv := range dynamicItems {
		vars[itemIndexVarName] = kv[0]
		vars[itemVarName] = kv[1]
		newDynVarVals, diag := parseDynVars(ctx, kv[0], kv[1], dynamic.items.ValueRange)
		if diags.Extend(diag) {
			return
		}
		nonDynamicContent, diag := applyDynamicContentVars(ctx, dynamic.children, newDataCtx, newDynVarVals)
		if diags.Extend(diag) {
			// stop dynamic block processing on error: it's likely that
			// the error will be repeated for each item and only add noise
			break
		}
		res = append(res, nonDynamicContent...)
	}
	return
}

const (
	itemIndexVarName = "dynamic_item_index"
	itemVarName      = "dynamic_item"
	itemNameVarName  = "item_name"
)

// getVarsCopy ensures a vars map exists in the context and returns it

func parseDynVars(ctx context.Context, idx, val plugindata.Data, rng hcl.Range, itemName ...string) (parsed *definitions.ParsedVars, diags diagnostics.Diag) {
	// use existing vars parser by creating a synthetic (dynamic_)vars block
	attrs := map[string]*hclsyntax.Attribute{
		itemIndexVarName: {
			Name: itemIndexVarName,
			Expr: &hclsyntax.LiteralValueExpr{
				Val: plugindata.Encapsulated.ValToCty(idx),
			},
			SrcRange:    rng,
			NameRange:   rng,
			EqualsRange: rng,
		},
		itemVarName: {
			Name: itemVarName,
			Expr: &hclsyntax.LiteralValueExpr{
				Val: plugindata.Encapsulated.ValToCty(val),
			},
			SrcRange:    rng,
			NameRange:   rng,
			EqualsRange: rng,
		},
	}

	// Add item_name if provided
	if len(itemName) > 0 {
		attrs[itemNameVarName] = &hclsyntax.Attribute{
			Name: itemNameVarName,
			Expr: &hclsyntax.LiteralValueExpr{
				Val: plugindata.Encapsulated.ValToCty(plugindata.String(itemName[0])),
			},
			SrcRange:    rng,
			NameRange:   rng,
			EqualsRange: rng,
		}
	}

	return parser.ParseVars(ctx, &hclsyntax.Block{
		Type:            "dynamic_vars",
		TypeRange:       rng,
		OpenBraceRange:  rng,
		CloseBraceRange: rng,
		Body: &hclsyntax.Body{
			SrcRange:   rng,
			EndRange:   rng,
			Attributes: attrs,
		},
	}, nil)
}
