package plugindata

import (
	"fmt"
	"reflect"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/blackstork-io/fabric/internal/utils/encapsulator"
	"github.com/blackstork-io/fabric/internal/utils"
)

var Encapsulated *encapsulator.Codec[Data]

func init() {
	Encapsulated = encapsulator.NewCodec("jq queriable", &encapsulator.CapsuleOps[Data]{
		GoString: func(v *Data) string {
			return fmt.Sprintf("%+v", *v)
		},
		TypeGoString: func(_ reflect.Type) string {
			return "JqQueryableType"
		},
		ConversionFrom: func(src cty.Type) func(*Data, cty.Path) (cty.Value, error) {
			return func(d *Data, p cty.Path) (cty.Value, error) {
				val, err := convert.Convert(pluginDataToCty(*d), src)
				if err != nil {
					err = p.NewError(err)
				}
				return val, err
			}
		},
		ConversionTo: func(dst cty.Type) func(cty.Value, cty.Path) (*Data, error) {
			return func(v cty.Value, p cty.Path) (*Data, error) {
				if !v.IsWhollyKnown() {
					return nil, p.NewErrorf("can't convert to jq-queriable: value is unknown")
				}
				data, err := ctyToPluginData(v)
				if err != nil {
					return nil, p.NewError(err)
				}
				return &data, nil
			}
		},
		RawEquals: func(a, b *Data) bool {
			return reflect.DeepEqual(*a, *b)
		},
	})
}

func ctyToPluginData(v cty.Value) (_ Data, err error) {
	if v.IsNull() {
		return nil, nil
	}
	ty := v.Type()
	switch {
	case ty.Equals(cty.Bool):
		return Bool(v.True()), nil
	case ty.Equals(cty.Number):
		f, _ := v.AsBigFloat().Float64()
		return Number(f), nil
	case ty.Equals(cty.String):
		return String(v.AsString()), nil
	case ty.IsTupleType() || ty.IsListType() || ty.IsSetType():
		list := make(List, v.LengthInt())
		i := 0
		for it := v.ElementIterator(); it.Next(); i++ {
			idx, val := it.Element()
			list[i], err = ctyToPluginData(val)
			if err != nil {
				if !ty.IsSetType() {
					err = cty.IndexPath(idx).NewError(err)
				}
				return
			}
		}
		return list, nil
	case ty.IsObjectType() || ty.IsMapType():
		m := make(Map, v.LengthInt())
		for it := v.ElementIterator(); it.Next(); {
			key, val := it.Element()
			keyStr := key.AsString()
			m[keyStr], err = ctyToPluginData(val)
			if err != nil {
				if ty.IsObjectType() {
					err = cty.GetAttrPath(keyStr).NewError(err)
				} else {
					err = cty.IndexPath(key).NewError(err)
				}
				return
			}
		}
		return m, nil
	case Encapsulated.CtyTypeEqual(ty):
		return *Encapsulated.MustFromCty(v), nil
	default:
		return nil, fmt.Errorf("can't convert to jq-queriable: type %s is unsupported", ty.FriendlyName())
	}
}

func pluginDataToCty(v Data) cty.Value {
	if v == nil {
		return cty.NullVal(cty.DynamicPseudoType)
	}
	v = v.AsPluginData()
	switch val := v.(type) {
	case nil:
		return cty.NullVal(cty.DynamicPseudoType)
	case Bool:
		return cty.BoolVal(bool(val))
	case Number:
		return cty.NumberFloatVal(float64(val))
	case String:
		return cty.StringVal(string(val))
	case List:
		return cty.TupleVal(utils.FnMap(val, pluginDataToCty))
	case Map:
		return cty.ObjectVal(utils.MapMap(val, pluginDataToCty))
	default:
		panic(fmt.Sprintf("unsupported Data type: %T", v))
	}
}
