package pluginapiv1

import (
	"fmt"
	"math/big"

	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

func panicToErr(r any) error {
	d := panicToDiag(r, "Failed to decode cty.Type")
	if d[0].Extra != nil {
		if e, ok := d[0].Extra.(*diagnostics.PathExtra); ok {
			return fmt.Errorf("%s: %s (%s)", d[0].Summary, d[0].Detail, e)
		}
	}
	return fmt.Errorf("%s: %s", d[0].Summary, d[0].Detail)
}

func decodeCtyType(src *CtyType) (_ cty.Type, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = panicToErr(r)
		}
	}()
	val := src.GetType()
	if val == nil {
		return cty.NilType, nil
	}
	_, ty := decodeCty(val, true)
	return ty, nil
}

func decodeCtyValue(src *CtyValue) (_ cty.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = panicToErr(r)
		}
	}()
	defer func() {
		if r := recover(); r != nil {
			d := panicToDiag(r, "Failed to decode cty.Value")
			err = fmt.Errorf("%s: %s", d[0].Summary, d[0].Detail)
		}
	}()
	val := src.GetValue()
	if val == nil {
		return cty.NilVal, nil
	}
	ctyVal, _ := decodeCty(val, false)
	return ctyVal, nil
}

// Returns the decoded sequence values or types (if sequence is empty).
// The second return value has length 1 for lists and sets, and varies for tuples.
func decodeSequence(seq *Cty_Sequence, decodeType bool) ([]cty.Value, []cty.Type) {
	i := -1
	onlyType := seq.GetOnlyType()
	if decodeType && !onlyType {
		panic(fmt.Errorf("attempted to decode a sequence of values as types"))
	}

	defer func() {
		if r := recover(); r != nil {
			var step cty.PathStep
			if i != -1 {
				if onlyType {
					step = cty.IndexStep{Key: cty.StringVal("<ctyDecoder empty item>")}
				} else {
					step = cty.IndexStep{Key: cty.NumberIntVal(int64(i))}
				}
			}
			traceback(r, step)
		}
	}()

	data := seq.GetData()
	var value *Cty
	if onlyType {
		types := make([]cty.Type, len(data))
		for i, value = range data {
			_, types[i] = decodeCty(value, true)
		}
		return nil, types
	} else {
		vals := make([]cty.Value, len(data))
		for i, value = range data {
			vals[i], _ = decodeCty(value, false)
		}
		return vals, nil
	}
}

func decodeCty(val *Cty, decodeType bool) (retVal cty.Value, retTy cty.Type) {
	switch data := val.GetData().(type) {
	case *Cty_Primitive_:
		switch primitive := data.Primitive.GetData().(type) {
		case *Cty_Primitive_Str:
			if decodeType {
				retTy = cty.String
			} else {
				retVal = cty.StringVal(primitive.Str)
			}
			return
		case *Cty_Primitive_Num:
			if decodeType {
				retTy = cty.Number
				return
			}
			var v big.Float
			err := v.GobDecode(primitive.Num)
			if err != nil {
				panic(fmt.Errorf("failed to decode a number: %w", err))
			}
			retVal = cty.NumberVal(&v)
			return
		case *Cty_Primitive_Bln:
			if decodeType {
				retTy = cty.Bool
			} else {
				retVal = cty.BoolVal(primitive.Bln)
			}
			return
		default:
			panic(fmt.Errorf("unsupported primitive value %T", val))
		}
	case *Cty_Object_:
		var name string
		defer func() {
			if r := recover(); r != nil {
				traceback(r, cty.GetAttrStep{Name: name})
			}
		}()
		obj := data.Object.GetData()
		if decodeType {
			attrs := make(map[string]cty.Type, len(obj))
			var v *Cty_Object_Attr
			var optional []string
			for name, v = range obj {
				if v.GetOptional() {
					optional = append(optional, name)
				}
				_, attrs[name] = decodeCty(v.GetData(), decodeType)
			}
			retTy = cty.ObjectWithOptionalAttrs(attrs, optional)
		} else {
			attrs := make(map[string]cty.Value, len(obj))
			var v *Cty_Object_Attr
			for name, v = range obj {
				attrs[name], _ = decodeCty(v.GetData(), decodeType)
			}
			retVal = cty.ObjectVal(attrs)
		}
		return
	case *Cty_Map:
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
		onlyType := data.Map.GetOnlyType()
		if decodeType && !onlyType {
			panic(fmt.Errorf("attempted to decode a mapping of values as types"))
		}
		values := data.Map.GetData()
		var value *Cty
		if decodeType || onlyType {
			key = "<ctyDecoder empty map type>"
			if len(values) != 1 {
				panic(fmt.Errorf("map has the wrong length: %d", len(values)))
			}
			for _, value = range values {
				_, elementTy := decodeCty(value, true)
				if decodeType {
					retTy = cty.Map(elementTy)
				} else {
					retVal = cty.MapValEmpty(elementTy)
				}
				return
			}
			panic("No values received in the map")
		} else {
			vals := make(map[string]cty.Value, len(values))
			for key, value = range values {
				vals[key], _ = decodeCty(value, decodeType)
			}
			retVal = cty.MapVal(vals)
			return
		}
	case *Cty_List:
		vals, types := decodeSequence(data.List, decodeType)
		if decodeType {
			retTy = cty.List(types[0])
		} else {
			if vals == nil {
				retVal = cty.ListValEmpty(types[0])
			} else {
				retVal = cty.ListVal(vals)
			}
		}
		return
	case *Cty_Set:
		vals, types := decodeSequence(data.Set, decodeType)
		if decodeType {
			retTy = cty.Set(types[0])
		} else {
			if vals == nil {
				retVal = cty.SetValEmpty(types[0])
			} else {
				retVal = cty.SetVal(vals)
			}
		}
		return
	case *Cty_Tuple:
		vals, types := decodeSequence(data.Tuple, decodeType)
		if decodeType {
			retTy = cty.Tuple(types)
		} else {
			retVal = cty.TupleVal(vals)
		}
		return
	case *Cty_Null:
		ty := data.Null.GetType()
		if ty == nil {
			panic(fmt.Errorf("typeless null encountered"))
		}
		_, ctyTy := decodeCty(ty, true)
		if decodeType {
			retTy = ctyTy
		} else {
			retVal = cty.NullVal(ctyTy)
		}
		return
	case *Cty_Caps:
		switch val := data.Caps.GetData().(type) {
		case *Cty_Capsule_PluginData:
			if decodeType {
				retTy = plugindata.Encapsulated.CtyType()
			} else {
				pluginData := decodeData(val.PluginData)
				retVal = plugindata.Encapsulated.ToCty(&pluginData)
			}
			return
		default:
			panic(fmt.Errorf("unsupported encapsulated value %T", val))
		}
	case *Cty_Unknown:
		if decodeType {
			panic(fmt.Errorf("encountered unknown value in type decoding context"))
		}
		ty := data.Unknown.GetType()
		if ty == nil {
			panic(fmt.Errorf("typeless unknown value encountered"))
		}
		_, ctyTy := decodeCty(ty, true)
		retVal = cty.UnknownVal(ctyTy)
		return
	case *Cty_Dyn:
		if decodeType {
			retTy = cty.DynamicPseudoType
		} else {
			// This is unreachable: cty.DynamicVal is exactly cty.UnknownVal(cty.DynamicPseudoType)
			// and would be decoded that way. Keeping for completeness
			retVal = cty.DynamicVal
		}
		return
	default:
		panic(fmt.Errorf("unsupported Cty %T", data))
	}
}
