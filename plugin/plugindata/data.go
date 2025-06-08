package plugindata

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"gopkg.in/yaml.v3"
)

type Data interface {
	Any() any
	data()
	Convertible
}

func (Number) data() {}
func (String) data() {}
func (Bool) data()   {}
func (Map) data()    {}
func (List) data()   {}
func (Time) data()   {}

func (d Number) AsPluginData() Data { return d }
func (d String) AsPluginData() Data { return d }
func (d Bool) AsPluginData() Data   { return d }
func (d Map) AsPluginData() Data    { return d }
func (d List) AsPluginData() Data   { return d }
func (d Time) AsPluginData() Data   { return d }

type Number float64

func (d Number) Any() any {
	return float64(d)
}

type String string

func (d String) Any() any {
	return string(d)
}

type Bool bool

func (d Bool) Any() any {
	return bool(d)
}

type Map map[string]Data

func (d Map) Any() any {
	dst := make(map[string]any, len(d))
	for k, v := range d {
		if v == nil {
			dst[k] = nil
			continue
		}
		dst[k] = v.Any()
	}
	return dst
}

func (d Map) SetWithPath(path []string, value Data) {

	leafKey := path[len(path)-1]
	branchKeys := path[0 : len(path)-1]

	submap := d

	for _, key := range branchKeys {
		if keyVal, exists := submap[key]; exists {
			submap = keyVal.(Map)
		} else {
			keyVal := make(Map)
			submap[key] = keyVal
			submap = keyVal
		}
	}
	submap[leafKey] = value
}

func (d Map) Clone() Map {
	dst := make(Map, len(d))
	for k, v := range d {
		switch v := v.(type) {
		case Map:
			dst[k] = v.Clone()
		case List:
			dst[k] = v.Clone()
		default:
			dst[k] = v
		}
	}
	return dst
}

type List []Data

func (d List) Any() any {
	dst := make([]any, len(d))
	for i, v := range d {
		if v == nil {
			dst[i] = nil
			continue
		}
		dst[i] = v.Any()
	}
	return dst
}

type Time time.Time

func (d Time) Any() any {
	return (time.Time)(d)
}

func (d List) Clone() List {
	dst := make(List, len(d))
	for i, v := range d {
		switch v := v.(type) {
		case Map:
			dst[i] = v.Clone()
		case List:
			dst[i] = v.Clone()
		default:
			dst[i] = v
		}
	}
	return dst
}

func UnmarshalJSON(data []byte) (Data, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return ParseAny(v)
}

func UnmarshalYAML(data []byte) (Data, error) {
	var v any
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return ParseAny(v)
}

func ParseAny(v any) (Data, error) {
	switch v := v.(type) {
	case nil:
		return nil, nil
	case bool:
		return Bool(v), nil
	case float64:
		return Number(v), nil
	case float32:
		return Number(v), nil
	case uint:
		return Number(v), nil
	case uint8:
		return Number(v), nil
	case uint16:
		return Number(v), nil
	case uint32:
		return Number(v), nil
	case uint64:
		return Number(v), nil
	case int:
		return Number(v), nil
	case int8:
		return Number(v), nil
	case int16:
		return Number(v), nil
	case int32:
		return Number(v), nil
	case int64:
		return Number(v), nil
	case uintptr:
		return Number(v), nil
	case string:
		return String(v), nil
	case time.Time:
		return Time(v), nil
	case []any:
		// this case would trigger only for []any
		dst := make(List, len(v))
		for i, e := range v {
			d, err := ParseAny(e)
			if err != nil {
				return nil, err
			}
			dst[i] = d
		}
		return dst, nil
	case map[string]any:
		return ParseMapAny(v)
	default:
		// Fallback to reflection for types not explicitly handled above.
		// This is where we solve the `[]string` vs `[]any` problem.
		val := reflect.ValueOf(v)
		switch val.Kind() {
		case reflect.Slice:
			// Handles any kind of slice, e.g., []string, []int, []CustomStruct
			dst := make(List, val.Len())
			for i := range val.Len() {
				// Get the element, convert it back to `any`, and recurse.
				elem := val.Index(i).Interface()
				parsedElem, err := ParseAny(elem)
				if err != nil {
					return nil, fmt.Errorf("error parsing slice element at index %d: %w", i, err)
				}
				dst[i] = parsedElem
			}
			return dst, nil
		case reflect.Map:
			// Handling any map with string keys, for example, `map[string]int` or `map[string]string`.
			if val.Type().Key().Kind() != reflect.String {
				return nil, fmt.Errorf("unsupported map key type: `%s` (only string keys are supported)", val.Type().Key())
			}

			dst := make(Map, val.Len())
			iter := val.MapRange()
			for iter.Next() {
				key := iter.Key().String()
				// Get the value, convert it back to `any`, and recurse.
				val := iter.Value().Interface()
				parsedVal, err := ParseAny(val)
				if err != nil {
					return nil, fmt.Errorf("error parsing map value for key %q: %w", key, err)
				}
				dst[key] = parsedVal
			}
			return dst, nil
		}
		return nil, fmt.Errorf("unsupported data type: `%T`", v)
	}
}

func ParseMapAny(v map[string]any) (Map, error) {
	if v == nil {
		return nil, nil
	}
	dst := make(Map)
	for k, e := range v {
		d, err := ParseAny(e)
		if err != nil {
			return nil, err
		}
		dst[k] = d
	}
	return dst, nil
}

type Convertible interface {
	AsPluginData() Data
}

func IsTruthy(d Data) (bool, error) {
	switch d := d.(type) {
	case Bool:
		return bool(d), nil
	case Number:
		return float64(d) != 0, nil
	case String:
		return string(d) != "", nil
	case List:
		return len(d) > 0, nil
	case Map:
		return len(d) > 0, nil
	case nil:
		return false, nil
	default:
		return false, fmt.Errorf("unsupported data type: `%T`", d)
	}
}
