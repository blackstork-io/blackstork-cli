package utils

import "github.com/zclconf/go-cty/cty"

func MapMapToCty(m map[string]map[string]cty.Value) (cty.Value) {
	res := make(map[string]cty.Value, len(m))
	for k, v := range m {
		if len(v) == 0 {
			continue
		}
		res[k] = cty.MapVal(v)
	}
	return cty.MapVal(res)
}

