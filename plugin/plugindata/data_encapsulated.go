// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package plugindata

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/pkg/utils/encapsulator"
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
				val, err := convert.Convert(PluginDataToCty(*d), src)
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
				data, err := CtyToPluginData(v)
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

func CtyToPluginData(v cty.Value) (_ Data, err error) {
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
			list[i], err = CtyToPluginData(val)
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
			m[keyStr], err = CtyToPluginData(val)
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

func CtyToPluginDataWithType(val cty.Value, typeName string) (Data, error) {
	// Handle nulls immediately
	if val.IsNull() || !val.IsKnown() {
		return nil, nil
	}

	normalizedType := strings.ToLower(typeName)

	switch normalizedType {
	case "string", "secret":
		// cty.Value -> string -> plugindata.String
		if val.Type() != cty.String {
			return nil, fmt.Errorf("expected string type for %s, got %s", typeName, val.Type().FriendlyName())
		}
		return String(val.AsString()), nil

	case "datetime":
		// Datetimes in cty are stored as strings. We must parse them to time.Time.
		if val.Type() != cty.String {
			return nil, fmt.Errorf("expected string for datetime, got %s", val.Type().FriendlyName())
		}

		_, err := time.Parse(time.RFC3339, val.AsString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse ISO8601 datetime '%s': %w", val.AsString(), err)
		}

		// return datetime as string, so it can be used in JQ queries
		return String(val.AsString()), nil

	case "bool":
		// cty.Value -> bool -> plugindata.Bool
		if val.Type() != cty.Bool {
			return nil, fmt.Errorf("expected bool, got %s", val.Type().FriendlyName())
		}
		return Bool(val.True()), nil

	case "int", "float", "number":
		// cty.Value (Number) -> float64 -> plugindata.Number
		if val.Type() != cty.Number {
			return nil, fmt.Errorf("expected number for %s, got %s", typeName, val.Type().FriendlyName())
		}

		// cty numbers are big.Float. We convert to float64 to satisfy plugindata.Number.
		f, _ := val.AsBigFloat().Float64()
		return Number(f), nil

	default:
		return nil, fmt.Errorf("unsupported target type: %s", typeName)
	}
}

func PluginDataToCty(v Data) cty.Value {
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
		return cty.TupleVal(utils.FnMap(val, PluginDataToCty))
	case Map:
		return cty.ObjectVal(utils.MapMap(val, PluginDataToCty))
		// 	case Time:
		// 		return cty.StringVal(time.Time(val).Format(time.RFC3339))
	default:
		panic(fmt.Sprintf("unsupported Data type: %T", v))
	}
}
