package dataspec

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/customdecode"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/utils"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/deferred"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

// EvaluateDeferred evaluates deferred values in the given attribute.
func EvaluateDeferred(ctx context.Context, dataCtx plugindata.Map, val *Attr) (res cty.Value, diags diagnostics.Diag) {
	if val == nil {
		return
	}
	res, diags = (&transformer{
		ctx:     ctx,
		dataCtx: dataCtx,
	}).transform(val.Value)
	diags.Refine(diagnostics.DefaultSubject(val.ValueRange))
	return
}

type transformer struct {
	ctx     context.Context
	dataCtx plugindata.Map
	path    cty.Path
}

func (t *transformer) transform(val cty.Value) (_ cty.Value, diags diagnostics.Diag) {
	ty := val.Type()
	var marks cty.ValueMarks
	val, marks = val.Unmark()
	switch {
	case ty.IsPrimitiveType():
		// nothing to do
	case ty.IsCapsuleType():
		switch {
		case deferred.Type.CtyTypeEqual(ty):
			// TODO: warn in these cases
			switch {
			case val.IsNull():
				slog.Debug("Null deferred value", "path", t.path.NewErrorf("path"))
				val = cty.NullVal(cty.DynamicPseudoType)
			case !val.IsKnown():
				slog.Debug("Unknown deferred value", "path", t.path.NewErrorf("path"))
				val = cty.UnknownVal(cty.DynamicPseudoType)
			default:
				eval := deferred.Type.MustFromCty(val)
				val, diags = eval.DeferredEval(t.ctx, t.dataCtx)
				diags.Refine(diagnostics.AddPath(t.path))
			}
		case plugindata.Encapsulated.CtyTypeEqual(ty):
			// nothing to do
		default:
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unexpected capsule value",
				Detail:   fmt.Sprintf("Capsule value %s is unexpected here", ty.FriendlyName()),
				Extra:    diagnostics.AddPath(t.path),
			})
		}

	case val.IsNull() || !val.IsKnown():
		// do nothing
	case ty.IsListType() || ty.IsSetType() || ty.IsTupleType():
		l := val.LengthInt()
		if l == 0 {
			break
		}
		vals := make([]cty.Value, 0, l)
		t.path = append(t.path, nil)
		for it := val.ElementIterator(); it.Next(); {
			key, subVal := it.Element()
			t.path[len(t.path)-1] = cty.IndexStep{
				Key: key,
			}
			subVal, diag := t.transform(subVal)
			diags.Extend(diag)
			vals = append(vals, subVal)
		}
		t.path = t.path[:len(t.path)-1]
		if diags.HasErrors() {
			break
		}
		switch {
		case ty.IsListType() && cty.CanListVal(vals):
			val = cty.ListVal(vals)
		case ty.IsSetType() && cty.CanSetVal(vals):
			val = cty.SetVal(vals)
		default:
			// Lists are replaced by tuples if no longer homogenous.
			// Sets are attempted to be converted to set type.
			val = cty.TupleVal(vals)
			if ty.IsSetType() {
				var err error
				val, err = convert.Convert(val, ty)
				if diags.AppendErr(err, "Failed to convert to set") {
					diags.Refine(diagnostics.AddPath(t.path))
				}
			}
		}
	case ty.IsMapType():
		l := val.LengthInt()
		if l == 0 {
			break
		}
		elems := make(map[string]cty.Value, l)
		t.path = append(t.path, nil)
		for it := val.ElementIterator(); it.Next(); {
			key, subVal := it.Element()
			t.path[len(t.path)-1] = cty.IndexStep{
				Key: key,
			}
			newVal, diag := t.transform(subVal)
			diags.Extend(diag)
			elems[key.AsString()] = newVal
		}
		t.path = t.path[:len(t.path)-1]
		if diags.HasErrors() {
			break
		}
		if cty.CanMapVal(elems) {
			val = cty.MapVal(elems)
		} else {
			val = cty.ObjectVal(elems)
		}
	case ty.IsObjectType():
		if ty.Equals(cty.EmptyObject) {
			break
		}
		atys := ty.AttributeTypes()
		elems := make(map[string]cty.Value, len(atys))
		t.path = append(t.path, nil)
		for name := range atys {
			subVal := val.GetAttr(name)
			t.path[len(t.path)-1] = cty.GetAttrStep{
				Name: name,
			}
			subVal, diag := t.transform(subVal)
			diags.Extend(diag)
			elems[name] = subVal
		}
		t.path = t.path[:len(t.path)-1]
		val = cty.ObjectVal(elems)
	default:
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unexpected value",
			Detail:   fmt.Sprintf("Value %s is unexpected here", ty.FriendlyName()),
			Extra:    diagnostics.AddPath(t.path),
		})
	}
	if diags.HasErrors() {
		val = cty.DynamicVal
	}
	return val.WithMarks(marks), diags
}

// Decodes hclsyntax.Block into a Block according to the given RootSpec.
// Basic validation is performed on the keys, values of attributes are not fully defined until deferred evaluation,
// so they are not type-checked in the resulting block.
// This function's result is expected to be processed later with EvalBlock to resolve the deferred values and validate everything.
func DecodeBlock(ctx context.Context, block *hclsyntax.Block, rootSpec *RootSpec) (res *Block, diags diagnostics.Diag) {
	return decodeBlock(block, rootSpec.BlockSpec(), fabctx.GetEvalContext(ctx))
}

// Decodes hclsyntax.Block into a Block according to the given RootSpec.
// Deferred evaluation is performed immediatly.
func DecodeAndEvalBlock(
	ctx context.Context,
	block *hclsyntax.Block,
	rootSpec *RootSpec,
	dataCtx plugindata.Map,
) (res *Block, diags diagnostics.Diag) {
	res, diags = decodeBlock(block, rootSpec.BlockSpec(), fabctx.GetEvalContext(ctx))
	if diags.HasErrors() {
		res = nil
		return
	}
	diags.Extend(EvalBlock(ctx, res, dataCtx))
	if diags.HasErrors() {
		res = nil
		return
	}
	return
}

// EvalBlockCopy evaluates deferred values in the given block and validates the attributes.
// The original block is not modified.
func EvalBlockCopy(
	ctx context.Context,
	block *Block,
	dataCtx plugindata.Map,
) (evaluatedBlock *Block, diags diagnostics.Diag) {
	if block == nil {
		return
	}
	blockC := *block
	evaluatedBlock = &blockC

	evaluatedBlock.Blocks = utils.FnMapDiags(&diags, block.Blocks, func(block *Block) (*Block, diagnostics.Diag) {
		return EvalBlockCopy(ctx, block, dataCtx)
	})
	evaluatedBlock.Attrs = utils.MapMapDiags(&diags, block.Attrs, func(attr *Attr) (*Attr, diagnostics.Diag) {
		attrC := *attr
		var diag diagnostics.Diag
		attrC.Value, diag = EvalAttr(ctx, attr, dataCtx)
		return &attrC, diag
	})
	return
}

// EvalBlock evaluates deferred values in the given block and validates the attributes.
// WARNING: This function modifies the input block (its attributes and blocks) in place.
// Not suitable for evaluating deferred blocks
func EvalBlock(ctx context.Context, block *Block, dataCtx plugindata.Map) (diags diagnostics.Diag) {
	if block == nil {
		return
	}
	for _, block := range block.Blocks {
		diags.Extend(EvalBlock(ctx, block, dataCtx))
	}
	for _, attr := range block.Attrs {
		val, diag := EvalAttr(ctx, attr, dataCtx)
		diags.Extend(diag)
		attr.Value = val
	}
	return
}

func decodeBlock(
	block *hclsyntax.Block,
	blockSpec *BlockSpec,
	ctx *hcl.EvalContext,
) (res *Block, diags diagnostics.Diag) {

	if block == nil {
		if blockSpec == nil {
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing a block and a block spec",
				Detail:   "A block and a blockspec are both missing, can't decode",
			})
			return
		}
		if blockSpec.Required {
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing a block",
				Detail:   fmt.Sprintf("Block of type %s is required", formatHeader(blockSpec.Header.AsDocLabels())),
			})
			return
		}
		name, labels := blockSpec.Header.AsDocLabels()
		labels = append(labels, "<default values>")
		block = &hclsyntax.Block{
			Type:   name,
			Labels: labels,
			Body:   &hclsyntax.Body{},
		}
	}

	res = &Block{
		Header:        make([]string, 1, len(block.Labels)+1),
		HeaderRanges:  make([]hcl.Range, 1, len(block.Labels)+1),
		Attrs:         make(Attributes, len(block.Body.Attributes)),
		Blocks:        make(Blocks, 0, len(block.Body.Blocks)),
		ContentsRange: block.Body.SrcRange,
	}

	res.Header[0] = block.Type
	res.Header = append(res.Header, block.Labels...)

	res.HeaderRanges[0] = block.TypeRange
	res.HeaderRanges = append(res.HeaderRanges, block.LabelRanges...)

	if blockSpec == nil {
		for _, blk := range block.Body.Blocks {
			b, diag := decodeBlock(blk, nil, ctx)
			res.Blocks = append(res.Blocks, b)
			diags.Extend(diag)
		}
		for k, attr := range block.Body.Attributes {
			val, diag := attr.Expr.Value(ctx)
			diags.Extend(diag)
			res.Attrs[k] = &Attr{
				Name:       attr.Name,
				NameRange:  attr.NameRange,
				Value:      val,
				ValueRange: attr.Expr.Range(),
			}
		}
		return
	}

	specWasUsed := make([]bool, len(blockSpec.Blocks))
nextBlock:
	for _, subBlock := range block.Body.Blocks {

		for i, subSpec := range blockSpec.Blocks {
			if !subSpec.Repeatable && specWasUsed[i] {
				continue
			}
			if !subSpec.Header.Match(subBlock.Type, subBlock.Labels) {
				continue
			}
			specWasUsed[i] = true
			b, diag := decodeBlock(subBlock, subSpec, ctx)
			diags.Extend(diag)
			res.Blocks = append(res.Blocks, b)
			continue nextBlock
		}

		if !blockSpec.AllowUnspecifiedBlocks {
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unexpected block found",
				Detail:   fmt.Sprintf("%s can not contain this block", formatHeader(block.Type, block.Labels)),
				Subject:  &subBlock.TypeRange,
				Context:  &block.Body.SrcRange,
			})
			continue nextBlock
		}

		b, diag := decodeBlock(subBlock, nil, ctx)
		res.Blocks = append(res.Blocks, b)
		diags.Extend(diag)
	}

	for i, subSpec := range blockSpec.Blocks {
		if subSpec.Required && !specWasUsed[i] {
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing a block",
				Detail: fmt.Sprintf(
					"%s requires a block of type %s to be defined",
					formatHeader(block.Type, block.Labels),
					formatHeader(subSpec.Header.AsDocLabels()),
				),
				Subject: block.Body.MissingItemRange().Ptr(),
			})
		}
	}

	attrs := maps.Clone(block.Body.Attributes)
	for _, spec := range blockSpec.Attrs {
		attr, found := utils.Pop(attrs, spec.Name)

		if !found {
			if spec.Constraints.Is(constraint.Required) {
				diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Missing required attribute",
					Detail:   fmt.Sprintf("The attribute %q is required, but no definition was found.", spec.Name),
					Subject:  block.Body.MissingItemRange().Ptr(),
				})
			} else if spec.DefaultVal != cty.NilVal {
				rng := block.DefRange()

				res.Attrs[spec.Name] = &Attr{
					Name:       spec.Name,
					NameRange:  rng,
					Value:      spec.DefaultVal,
					ValueRange: rng,
					Secret:     spec.Secret,
					spec:       spec,
				}
			}
			continue
		}

		if spec.Deprecated != "" {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Deprecated attribute",
				Detail:   fmt.Sprintf("The attribute %q is deprecated: %s", spec.Name, spec.Deprecated),
				Subject:  &attr.NameRange,
				Context:  &attr.SrcRange,
			})
		}

		decAttr, diag := DecodeAttr(ctx, attr, spec)
		if diags.Extend(diag) {
			continue
		}
		res.Attrs[spec.Name] = decAttr
	}

	for name, attr := range attrs {
		slog.Warn("ATTR", "name", name, "attr", attr, "block", block)
		if !blockSpec.AllowUnspecifiedAttributes {
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Provided attribute is not supported",
				Detail:   fmt.Sprintf("Unsupported attribute `%q` encountered", name),
				Subject:  &attr.NameRange,
				Context:  hcl.RangeBetween(attr.NameRange, attr.Expr.Range()).Ptr(),
			})
			continue
		}
		decAttr, diag := DecodeAttr(ctx, attr, nil)
		if diags.Extend(diag) {
			continue
		}
		res.Attrs[name] = decAttr
	}
	return
}

// Decodes hclsyntax.Attribute into a cty.Value according to the given AttrSpec.
// No validation is performed on the value.
func DecodeAttr(ctx *hcl.EvalContext, attr *hclsyntax.Attribute, spec *AttrSpec) (val *Attr, diags diagnostics.Diag) {
	var (
		decodeFn customdecode.CustomExpressionDecoderFunc
		value    cty.Value
		diag     hcl.Diagnostics
		secret   bool
	)
	if spec != nil {
		decodeFn = customdecode.CustomExpressionDecoderForType(spec.Type)
		secret = spec.Secret
	}
	if decodeFn != nil {
		value, diag = decodeFn(attr.Expr, ctx)
	} else {
		value, diag = attr.Expr.Value(ctx)
	}
	diags = diagnostics.Diag(diag).Refine(diagnostics.DefaultSubject(attr.Expr.Range()))
	if diags.HasErrors() {
		return
	}
	val = &Attr{
		Name:       attr.Name,
		NameRange:  attr.NameRange,
		Value:      value,
		ValueRange: attr.Expr.Range(),
		Secret:     secret,
		spec:       spec,
	}

	return
}

// EvalAttr evaluates deferred values in the given attribute and validates it.
func EvalAttr(ctx context.Context, attr *Attr, dataCtx plugindata.Map) (val cty.Value, diags diagnostics.Diag) {
	val, diag := EvaluateDeferred(ctx, dataCtx, attr)
	if diags.Extend(diag) {
		return
	}
	if attr.spec == nil {
		return // can't convert or validate further
	}
	// convert
	var err error
	val, err = convert.Convert(val, attr.spec.Type)
	if err != nil {
		diags.Extend(diagnostics.FromErr(
			err,
			diagnostics.DefaultSummary("Incorrect attribute value type"),
			diagnostics.DefaultSubject(attr.ValueRange),
		))
		return
	}
	if attr.spec.Constraints.Is(constraint.TrimSpace) {
		val = trimSpace(val)
	}
	diags.Extend(attr.spec.ValidateValue(val).Refine(diagnostics.DefaultSubject(attr.ValueRange)))
	return
}

// DecodeAndEvalAttr decodes hclsyntax.Attribute into a Attr according to the given AttrSpec and evaluates it.
func DecodeAndEvalAttr(
	ctx context.Context,
	hclAttr *hclsyntax.Attribute,
	spec *AttrSpec,
	dataCtx plugindata.Map,
) (val cty.Value, diags diagnostics.Diag) {
	evalCtx := fabctx.GetEvalContext(ctx)
	attr, diags := DecodeAttr(evalCtx, hclAttr, spec)
	if diags.HasErrors() {
		return
	}
	val, diags = EvalAttr(ctx, attr, dataCtx)
	return
}
