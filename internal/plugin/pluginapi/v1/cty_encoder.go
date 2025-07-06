package pluginapiv1

import (
	"fmt"
	"slices"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

type panicErr struct {
	p    any
	path cty.Path
}

func traceback(r any, step cty.PathStep) {
	err, ok := r.(*panicErr)
	if !ok {
		err = &panicErr{p: r}
	}
	if step != nil {
		err.path = append(err.path, step)
	}
	panic(err)
}

func panicToDiag(r any, summary string) diagnostics.Diag {
	var extra any
	var detail string
	panicErrVal, ok := r.(*panicErr)
	if ok {
		slices.Reverse(panicErrVal.path)
		extra = diagnostics.AddPath(panicErrVal.path)
	} else if err, ok := r.(error); ok {
		detail = err.Error()
	} else {
		detail = fmt.Sprintf("Unexpected panic: %v", r)
	}

	return diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  summary,
		Detail:   detail,
		Extra:    extra,
	}}
}

func encodeCtyType(ty cty.Type) (res *CtyType, diags diagnostics.Diag) {
	defer func() {
		if r := recover(); r != nil {
			diags = panicToDiag(r, "Failed to encode cty.Type")
		}
	}()
	res = &CtyType{
		Type: encodeCty(cty.NilVal, ty),
	}
	return
}

func encodeCtyValue(val cty.Value) (res *CtyValue, diags diagnostics.Diag) {
	defer func() {
		if r := recover(); r != nil {
			diags = panicToDiag(r, "Failed to encode cty.Value")
		}
	}()
	res = &CtyValue{
		Value: encodeCty(val, cty.NilType),
	}
	return
}

// encodeCty encodes given cty.Value and/or cty.Type into Cty message.
// If value is cty.NilVal, the resulting Cty message must be used in CtyType.
// If ty is cty.NilType, the resulting Cty message must be used in CtyValue.
// If both are nil - the resulting Cty will be nil and would be decoded as cty.NilVal or cty.NilType
// depending on the encapsulating message.
// If both are present - it is assumed that ty is the same as val.Type().
func encodeCty(val cty.Value, ty cty.Type) *Cty {
	if ty == cty.NilType {
		ty = val.Type()
		// Both were nil, it's a cty.NilVal or cty.NilType
		if ty == cty.NilType {
			return nil
		}
	}
	if val != cty.NilVal {
		val, _ = val.Unmark()
	}
	hasVal := val != cty.NilVal

	if hasVal {
		// encode special types
		if val.IsNull() {
			return &Cty{
				Data: &Cty_Null{
					Null: &CtyType{
						Type: encodeCty(cty.NilVal, ty),
					},
				},
			}
		}
		if !val.IsKnown() {
			return &Cty{
				Data: &Cty_Unknown{
					Unknown: &CtyType{
						Type: encodeCty(cty.NilVal, ty),
					},
				},
			}
		}
	}
	switch {
	case ty == cty.DynamicPseudoType:
		return &Cty{
			Data: &Cty_Dyn{
				Dyn: &Cty_Dynamic{},
			},
		}

	case ty.IsPrimitiveType():
		var primitive isCty_Primitive_Data
		switch ty {
		case cty.String:
			var data string
			if hasVal {
				data = val.AsString()
			}
			primitive = &Cty_Primitive_Str{
				Str: data,
			}
		case cty.Number:
			var data []byte
			if hasVal {
				var err error
				data, err = val.AsBigFloat().GobEncode()
				if err != nil {
					panic(fmt.Errorf("failed to encode a number: %w", err))
				}
			}
			primitive = &Cty_Primitive_Num{
				Num: data,
			}
		case cty.Bool:
			var data bool
			if hasVal {
				data = val.True()
			}
			primitive = &Cty_Primitive_Bln{
				Bln: data,
			}
		}
		return &Cty{
			Data: &Cty_Primitive_{
				Primitive: &Cty_Primitive{
					Data: primitive,
				},
			},
		}

	case ty.IsObjectType():
		var name string
		defer func() {
			if r := recover(); r != nil {
				traceback(r, cty.GetAttrStep{Name: name})
			}
		}()
		attrTypes := ty.AttributeTypes()
		data := make(map[string]*Cty_Object_Attr, len(attrTypes))
		optional := ty.OptionalAttributes()
		attrVal := cty.NilVal
		var attrTy cty.Type
		for name, attrTy = range attrTypes {
			// correct even if optional is a nil map
			_, isOptional := optional[name]
			if hasVal {
				attrVal = val.GetAttr(name)
			}
			data[name] = &Cty_Object_Attr{
				Data:     encodeCty(attrVal, attrTy),
				Optional: isOptional,
			}
		}
		return &Cty{
			Data: &Cty_Object_{
				Object: &Cty_Object{
					Data: data,
				},
			},
		}

	case ty.IsMapType():
		var key string
		defer func() {
			if r := recover(); r != nil {
				var step cty.PathStep
				if key != "" {
					step = cty.IndexStep{Key: cty.StringVal(key)}
				}
				traceback(r, step)
			}
		}()
		mapTy := ty.ElementType()
		var mapping Cty_Mapping
		mapping.OnlyType = !hasVal || val.LengthInt() == 0
		if mapping.OnlyType {
			key = "<ctyEncoder empty map type>"
			mapping.Data = map[string]*Cty{
				"": encodeCty(cty.NilVal, mapTy),
			}
		} else {
			ctyVals := val.AsValueMap()
			mapping.Data = make(map[string]*Cty, len(ctyVals))
			var ctyVal cty.Value
			for key, ctyVal = range ctyVals {
				mapping.Data[key] = encodeCty(ctyVal, mapTy)
			}
		}
		return &Cty{
			Data: &Cty_Map{
				Map: &mapping,
			},
		}

	case ty.IsListType() || ty.IsSetType() || ty.IsTupleType():
		i := -1
		var encodingEmptySeq bool
		defer func() {
			if r := recover(); r != nil {
				var step cty.PathStep
				if encodingEmptySeq {
					step = cty.IndexStep{Key: cty.StringVal("<ctyEncoder empty item>")}
				} else if i != -1 {
					step = cty.IndexStep{Key: cty.NumberIntVal(int64(i))}
				}
				traceback(r, step)
			}
		}()

		var seq Cty_Sequence

		seq.OnlyType = !hasVal || (!ty.IsTupleType() && val.LengthInt() == 0)
		if seq.OnlyType {
			if ty.IsTupleType() {
				ctyTypes := ty.TupleElementTypes()
				seq.Data = make([]*Cty, len(ctyTypes))
				var ctyTy cty.Type
				for i, ctyTy = range ctyTypes {
					seq.Data[i] = encodeCty(cty.NilVal, ctyTy)
				}
			} else {
				encodingEmptySeq = true
				seq.Data = []*Cty{encodeCty(cty.NilVal, ty.ElementType())}
			}
		} else {
			ctyVals := val.AsValueSlice()
			seq.Data = make([]*Cty, len(ctyVals))
			var ctyVal cty.Value
			for i, ctyVal = range ctyVals {
				seq.Data[i] = encodeCty(ctyVal, cty.NilType)
			}
		}

		switch {
		case ty.IsListType():
			return &Cty{
				Data: &Cty_List{
					List: &seq,
				},
			}
		case ty.IsSetType():
			return &Cty{
				Data: &Cty_Set{
					Set: &seq,
				},
			}
		case ty.IsTupleType():
			return &Cty{
				Data: &Cty_Tuple{
					Tuple: &seq,
				},
			}
		default:
			panic(fmt.Errorf("unxpected type (not a list, set or tuple: %q", ty.FriendlyName()))
		}
	case ty.IsCapsuleType():
		if !plugindata.Encapsulated.CtyTypeEqual(ty) {
			panic(fmt.Errorf("unsupported capsule type: %q", ty.FriendlyName()))
		}
		var data plugindata.Data
		if hasVal {
			dataPtr, err := plugindata.Encapsulated.FromCty(val)
			if err != nil {
				panic(fmt.Errorf("failed to decode capsule type: %w", err))
			}
			if dataPtr != nil {
				data = *dataPtr
			}
		}
		return &Cty{
			Data: &Cty_Caps{
				Caps: &Cty_Capsule{
					Data: &Cty_Capsule_PluginData{
						PluginData: encodeData(data),
					},
				},
			},
		}
	}
	panic(fmt.Errorf("unsupported type: %q", ty.FriendlyName()))
}
