// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package definitions

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
)

func TestInputBlockParseDefaultValueJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   cty.Value
		schema  cty.Value
		want    any
		wantErr string
	}{
		{
			name:   "array",
			value:  cty.TupleVal([]cty.Value{cty.StringVal("first"), cty.StringVal("second")}),
			schema: cty.NullVal(cty.DynamicPseudoType),
			want:   []any{"first", "second"},
		},
		{
			name: "object",
			value: cty.ObjectVal(map[string]cty.Value{
				"enabled": cty.BoolVal(true),
				"limit":   cty.NumberIntVal(5),
			}),
			schema: cty.NullVal(cty.DynamicPseudoType),
			want: map[string]any{
				"enabled": true,
				"limit":   float64(5),
			},
		},
		{
			name:  "schema accepts value",
			value: cty.TupleVal([]cty.Value{cty.StringVal("item")}),
			schema: cty.ObjectVal(map[string]cty.Value{
				"type": cty.StringVal("array"),
				"items": cty.ObjectVal(map[string]cty.Value{
					"type": cty.StringVal("string"),
				}),
			}),
			want: []any{"item"},
		},
		{
			name:  "schema rejects value",
			value: cty.TupleVal([]cty.Value{cty.NumberIntVal(1)}),
			schema: cty.ObjectVal(map[string]cty.Value{
				"type": cty.StringVal("array"),
				"items": cty.ObjectVal(map[string]cty.Value{
					"type": cty.StringVal("string"),
				}),
			}),
			wantErr: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := &InputBlock{
				Type:         "json",
				DefaultValue: tt.value,
				Schema:       tt.schema,
			}

			got, err := input.ParseDefaultValue()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got.Any())
		})
	}
}

func TestInputBlockParseDefaultValuePrimitive(t *testing.T) {
	t.Parallel()

	input := &InputBlock{
		Type:         "bool",
		DefaultValue: cty.BoolVal(true),
		Schema:       cty.NullVal(cty.DynamicPseudoType),
	}

	got, err := input.ParseDefaultValue()
	require.NoError(t, err)
	require.Equal(t, plugindata.Bool(true), got)
}
