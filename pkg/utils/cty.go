// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package utils

import "github.com/zclconf/go-cty/cty"

func MapMapToCty(m map[string]map[string]cty.Value) cty.Value {
	res := make(map[string]cty.Value, len(m))
	for k, v := range m {
		if len(v) == 0 {
			continue
		}
		res[k] = cty.MapVal(v)
	}
	return cty.MapVal(res)
}
