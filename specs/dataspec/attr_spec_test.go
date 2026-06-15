// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package dataspec_test

import (
	"context"
	"testing"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func TestValidation(t *testing.T) {
	// t.Parallel()
	for _, tc := range []struct {
		name        string
		obj         *dataspec.AttrSpec
		inputVal    cty.Value
		expectedVal cty.Value
		asserts     diagtest.Asserts
	}{
		{
			name: "basicAttribute",
			obj: &dataspec.AttrSpec{
				Name: "test",
				Type: cty.String,
			},
			inputVal:    cty.NullVal(cty.String),
			expectedVal: cty.NullVal(cty.String),
		},
		{
			name: "basicAttributeMissing",
			obj: &dataspec.AttrSpec{
				Name: "test",
				Type: cty.String,
			},
			inputVal:    cty.NilVal,
			expectedVal: cty.NilVal,
		},
		{
			name: "requiredAttributeMissing",
			obj: &dataspec.AttrSpec{
				Name:        "test",
				Type:        cty.String,
				ExampleVal:  cty.StringVal("test"),
				Constraints: constraint.Required,
			},
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("Missing required attribute"),
			}},
		},
		{
			name: "requiredAttributeMissingNonnil",
			obj: &dataspec.AttrSpec{
				Name:        "test",
				Type:        cty.String,
				ExampleVal:  cty.StringVal("test"),
				Constraints: constraint.Required | constraint.NonNull,
			},
			asserts: diagtest.Asserts{
				{
					diagtest.IsError,
					diagtest.SummaryContains("Missing required attribute"),
				},
			},
		},
		{
			name: "Null_for_Nonnull_value",
			obj: &dataspec.AttrSpec{
				Name:        "test",
				Type:        cty.String,
				ExampleVal:  cty.StringVal("test"),
				Constraints: constraint.Required | constraint.NonNull,
			},
			inputVal: cty.NullVal(cty.String),
			asserts: diagtest.Asserts{
				{
					diagtest.IsError,
					diagtest.SummaryContains("Attribute must be non-null"),
				},
			},
		},
		{
			name: "empty_for_nonnull_value",
			obj: &dataspec.AttrSpec{
				Name:        "test",
				Type:        cty.String,
				ExampleVal:  cty.StringVal("test"),
				Constraints: constraint.Required | constraint.NonNull,
			},
			inputVal:    cty.StringVal(""),
			expectedVal: cty.StringVal(""),
		},
		{
			name: "Nonempty",
			obj: &dataspec.AttrSpec{
				Name:        "test",
				Type:        cty.String,
				ExampleVal:  cty.StringVal("test"),
				Constraints: constraint.Required | constraint.NonEmpty,
			},
			inputVal:    cty.StringVal(""),
			expectedVal: cty.StringVal(""),
			asserts: diagtest.Asserts{
				{
					diagtest.IsError,
					diagtest.SummaryContains("Attribute must be non-empty"),
				},
			},
		},
		{
			name: "Nonempty_spaces",
			obj: &dataspec.AttrSpec{
				Name:        "test",
				Type:        cty.String,
				ExampleVal:  cty.StringVal("test"),
				Constraints: constraint.Required | constraint.NonEmpty,
			},
			inputVal:    cty.StringVal("    "),
			expectedVal: cty.StringVal("    "),
		},
		{
			name: "Trimmed_Nonempty_spaces",
			obj: &dataspec.AttrSpec{
				Name:        "test",
				Type:        cty.String,
				ExampleVal:  cty.StringVal("test"),
				Constraints: constraint.Required | constraint.TrimmedNonEmpty,
			},
			inputVal: cty.StringVal("   "),
			asserts: diagtest.Asserts{
				{
					diagtest.IsError,
					diagtest.SummaryContains("Attribute must be non-empty"),
				},
			},
		},
		{
			name: "Length_check_min",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.String,
				ExampleVal:   cty.StringVal("qwertyqwerty"),
				Constraints:  constraint.RequiredMeaningful,
				MinInclusive: cty.NumberIntVal(10),
			},
			inputVal: cty.StringVal("hello"),
			asserts: diagtest.Asserts{
				{
					diagtest.IsError,
					diagtest.SummaryContains("Attribute length is not in range"),
					diagtest.DetailContains(">=", "10"),
				},
			},
		},
		{
			name: "Length_check_correct_utf8_multibyte",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.String,
				DefaultVal:   cty.StringVal(";"),
				MinInclusive: cty.NumberIntVal(1),
				MaxInclusive: cty.NumberIntVal(1),
			},
			inputVal:    cty.StringVal("😭"),
			asserts:     diagtest.Asserts{},
			expectedVal: cty.StringVal("😭"),
		},
		{
			name: "Length_check_max",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.String,
				ExampleVal:   cty.StringVal("12"),
				Constraints:  constraint.Required,
				MaxInclusive: cty.NumberIntVal(2),
			},
			inputVal: cty.StringVal("hello"),
			asserts: diagtest.Asserts{
				{
					diagtest.IsError,
					diagtest.SummaryContains("Attribute length is not in range"),
					diagtest.DetailContains("<=", "2"),
				},
			},
		},
		{
			name: "Length_check_range",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.String,
				ExampleVal:   cty.StringVal("t"),
				Constraints:  constraint.RequiredMeaningful,
				MinInclusive: cty.NumberIntVal(1),
				MaxInclusive: cty.NumberIntVal(3),
			},
			inputVal: cty.StringVal("hello"),
			asserts: diagtest.Asserts{
				{
					diagtest.IsError,
					diagtest.SummaryContains("Attribute length is not in range"),
					diagtest.DetailContains("1", "3"),
				},
			},
		},
		{
			name: "Length_check_range_ok",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.String,
				ExampleVal:   cty.StringVal("test1"),
				Constraints:  constraint.RequiredMeaningful,
				MinInclusive: cty.NumberIntVal(5),
				MaxInclusive: cty.NumberIntVal(5),
			},
			inputVal:    cty.StringVal("hello"),
			expectedVal: cty.StringVal("hello"),
		},
		{
			name: "Length_check_range_exact",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.String,
				Constraints:  constraint.RequiredMeaningful,
				ExampleVal:   cty.StringVal("test1"),
				MinInclusive: cty.NumberIntVal(5),
				MaxInclusive: cty.NumberIntVal(5),
			},
			inputVal: cty.StringVal("hello_123"),
			asserts: diagtest.Asserts{
				{
					diagtest.IsError,
					diagtest.SummaryContains("Attribute length is not in range"),
					diagtest.DetailContains("exactly", "5"),
				},
			},
		},
		{
			name: "Length_inverted",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.String,
				ExampleVal:   cty.StringVal("t"),
				Constraints:  constraint.RequiredMeaningful,
				MinInclusive: cty.NumberIntVal(10),
				MaxInclusive: cty.NumberIntVal(1),
			},
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("MinInclusive must be <= MaxInclusive"),
			}},
			inputVal: cty.StringVal("hello_123"),
		},

		{
			name: "MinLength_<0",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.String,
				ExampleVal:   cty.StringVal("t"),
				Constraints:  constraint.RequiredMeaningful,
				MinInclusive: cty.NumberIntVal(-1),
			},
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("MinInclusive", "must be >= 0"),
			}},
			inputVal: cty.StringVal("hello_123"),
		},
		{
			name: "MaxLength_<0",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.String,
				ExampleVal:   cty.StringVal("t"),
				Constraints:  constraint.RequiredMeaningful,
				MaxInclusive: cty.NumberIntVal(-1),
			},
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("MaxInclusive", "must be >= 0"),
			}},
			inputVal: cty.StringVal("hello_123"),
		},
		{
			name: "oneof",
			obj: &dataspec.AttrSpec{
				Name:        "test",
				Type:        cty.String,
				ExampleVal:  cty.StringVal("a"),
				Constraints: constraint.RequiredMeaningful,
				OneOf: constraint.OneOf{
					cty.StringVal("a"),
					cty.StringVal("b"),
					cty.StringVal("c"),
				},
			},
			inputVal:    cty.StringVal("a"),
			expectedVal: cty.StringVal("a"),
		},
		{
			name: "oneof_err",
			obj: &dataspec.AttrSpec{
				Name:        "test",
				Type:        cty.String,
				ExampleVal:  cty.StringVal("a"),
				Constraints: constraint.RequiredMeaningful,
				OneOf: constraint.OneOf{
					cty.StringVal("a"),
					cty.StringVal("b"),
					cty.StringVal("c"),
				},
			},
			inputVal: cty.StringVal("x"),
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("not one of"),
				diagtest.DetailContains(`"a", "b", "c"`),
			}},
		},
		{
			name: "number_min",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.Number,
				ExampleVal:   cty.NumberIntVal(1),
				Constraints:  constraint.RequiredMeaningful,
				MinInclusive: cty.NumberFloatVal(0.5),
			},
			inputVal:    cty.NumberIntVal(1),
			expectedVal: cty.NumberIntVal(1),
		},
		{
			name: "number_min_err",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.Number,
				ExampleVal:   cty.NumberIntVal(1),
				Constraints:  constraint.RequiredMeaningful,
				MinInclusive: cty.NumberFloatVal(0.5),
			},
			inputVal: cty.NumberIntVal(0),
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("Attribute is not in range"),
				diagtest.DetailContains(`>=`),
			}},
		},
		{
			name: "number_max",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.Number,
				ExampleVal:   cty.NumberIntVal(1),
				Constraints:  constraint.RequiredMeaningful,
				MaxInclusive: cty.NumberFloatVal(2.5),
			},
			inputVal:    cty.NumberFloatVal(2.3),
			expectedVal: cty.NumberFloatVal(2.3),
		},
		{
			name: "number_max_err",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.Number,
				ExampleVal:   cty.NumberIntVal(1),
				Constraints:  constraint.RequiredMeaningful,
				MaxInclusive: cty.NumberFloatVal(2.5),
			},
			inputVal: cty.NumberFloatVal(2.7),
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("Attribute is not in range"),
			}},
		},
		{
			name: "number_range",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.Number,
				ExampleVal:   cty.NumberIntVal(2),
				Constraints:  constraint.RequiredMeaningful,
				MinInclusive: cty.NumberFloatVal(1.5),
				MaxInclusive: cty.NumberFloatVal(2.5),
			},
			inputVal:    cty.NumberFloatVal(1.7),
			expectedVal: cty.NumberFloatVal(1.7),
		},
		{
			name: "number_range_err",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.Number,
				ExampleVal:   cty.NumberIntVal(2),
				Constraints:  constraint.RequiredMeaningful,
				MinInclusive: cty.NumberFloatVal(1.5),
				MaxInclusive: cty.NumberFloatVal(2.5),
			},
			inputVal: cty.NumberFloatVal(4.2),
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("Attribute is not in range"),
			}},
		},
		{
			name: "length_check",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.List(cty.Number),
				Constraints:  constraint.RequiredMeaningful,
				ExampleVal:   cty.ListVal([]cty.Value{cty.NumberIntVal(1)}),
				MinInclusive: cty.NumberIntVal(1),
				MaxInclusive: cty.NumberIntVal(2),
			},
			inputVal:    cty.ListVal([]cty.Value{cty.NumberIntVal(1)}),
			expectedVal: cty.ListVal([]cty.Value{cty.NumberIntVal(1)}),
			asserts:     diagtest.Asserts{},
		},
		{
			name: "length_check_2",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.List(cty.Number),
				Constraints:  constraint.RequiredMeaningful,
				ExampleVal:   cty.ListVal([]cty.Value{cty.NumberIntVal(1)}),
				MinInclusive: cty.NumberIntVal(1),
				MaxInclusive: cty.NumberIntVal(2),
			},
			inputVal:    cty.ListVal([]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2)}),
			expectedVal: cty.ListVal([]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2)}),
			asserts:     diagtest.Asserts{},
		},
		{
			name: "length_check_err",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.List(cty.Number),
				Constraints:  constraint.RequiredMeaningful,
				ExampleVal:   cty.ListVal([]cty.Value{cty.NumberIntVal(1)}),
				MinInclusive: cty.NumberIntVal(1),
				MaxInclusive: cty.NumberIntVal(2),
			},
			inputVal: cty.ListVal([]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2), cty.NumberIntVal(3)}),
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("length", "not in range"),
			}},
		},
		{
			name: "length_check_constraint_err",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.List(cty.Number),
				Constraints:  constraint.RequiredMeaningful,
				ExampleVal:   cty.ListVal([]cty.Value{cty.NumberIntVal(1)}),
				MinInclusive: cty.NumberIntVal(-1),
				MaxInclusive: cty.NumberFloatVal(1.5),
			},
			inputVal: cty.ListVal([]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2), cty.NumberIntVal(3)}),
			asserts: diagtest.Asserts{
				{
					diagtest.IsError,
					diagtest.SummaryContains("MinInclusive", ">= 0"),
				},
				{
					diagtest.IsError,
					diagtest.SummaryContains("MaxInclusive", "must be an integer"),
				},
			},
		},
		{
			name: "length_check_constraint_err2",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.List(cty.Number),
				Constraints:  constraint.RequiredMeaningful,
				ExampleVal:   cty.ListVal([]cty.Value{cty.NumberIntVal(1)}),
				MinInclusive: cty.NumberFloatVal(0.5),
				MaxInclusive: cty.NumberIntVal(2),
			},
			inputVal: cty.ListVal([]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2), cty.NumberIntVal(3)}),
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("must be an integer"),
			}},
		},
		{
			name: "length_check_constraint_err3",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.List(cty.Number),
				Constraints:  constraint.RequiredMeaningful,
				ExampleVal:   cty.ListVal([]cty.Value{cty.NumberIntVal(1)}),
				MinInclusive: cty.NumberIntVal(1),
				MaxInclusive: cty.NumberIntVal(0),
			},
			inputVal: cty.ListVal([]cty.Value{cty.NumberIntVal(1), cty.NumberIntVal(2), cty.NumberIntVal(3)}),
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("MinInclusive must be <= MaxInclusive"),
			}},
		},
		{
			name: "deprecation",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.Number,
				ExampleVal:   cty.NumberIntVal(2),
				Constraints:  constraint.RequiredMeaningful,
				MinInclusive: cty.NumberFloatVal(1.5),
				MaxInclusive: cty.NumberFloatVal(2.5),
				Deprecated:   "test deprecation message",
			},
			inputVal: cty.NumberFloatVal(4.2),
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("Attribute is not in range"),
			}, {
				diagtest.IsWarning,
				diagtest.SummaryContains("Deprecated"),
				diagtest.DetailContains("test deprecation message"),
			}},
		},
		{
			name: "integer_constraint",
			obj: &dataspec.AttrSpec{
				Name:        "test",
				Type:        cty.Number,
				ExampleVal:  cty.NumberIntVal(2),
				Constraints: constraint.RequiredMeaningful | constraint.Integer,
			},
			inputVal:    cty.NumberFloatVal(4),
			expectedVal: cty.NumberFloatVal(4),
			asserts:     diagtest.Asserts{},
		},
		{
			name: "integer_constraint_violated",
			obj: &dataspec.AttrSpec{
				Name:        "test",
				Type:        cty.Number,
				ExampleVal:  cty.NumberIntVal(2),
				Constraints: constraint.RequiredMeaningful | constraint.Integer,
			},
			inputVal: cty.NumberFloatVal(4.3),
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("must be an integer"),
			}},
		},
		{
			name: "incorrect_example",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.Number,
				ExampleVal:   cty.NumberIntVal(2),
				Constraints:  constraint.RequiredMeaningful,
				MaxInclusive: cty.NumberFloatVal(1),
			},
			inputVal: cty.NumberFloatVal(0),
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("Example value", "not in range"),
			}},
		},
		{
			name: "incorrect_default",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.Number,
				DefaultVal:   cty.NumberIntVal(2),
				Constraints:  constraint.Meaningful,
				MaxInclusive: cty.NumberFloatVal(1),
			},
			asserts: diagtest.Asserts{{
				diagtest.IsError,
				diagtest.SummaryContains("Default value", "not in range"),
			}},
		},
		{
			name: "default_and_example_on_required",
			obj: &dataspec.AttrSpec{
				Name:        "test",
				Type:        cty.Number,
				DefaultVal:  cty.NumberIntVal(2),
				Constraints: constraint.Required,
			},
			asserts: diagtest.Asserts{
				{
					diagtest.IsWarning,
					diagtest.SummaryContains("Missing example value"),
				},
				{
					diagtest.IsError,
					diagtest.SummaryContains("Default value is specified"),
				},
			},
		},
		{
			name: "default_value",
			obj: &dataspec.AttrSpec{
				Name:       "test",
				Type:       cty.Number,
				DefaultVal: cty.NumberIntVal(2),
			},
			expectedVal: cty.NumberIntVal(2),
		},
		{
			name: "default_value_null",
			obj: &dataspec.AttrSpec{
				Name:       "test",
				Type:       cty.Number,
				DefaultVal: cty.NumberIntVal(2),
			},
			inputVal:    cty.NullVal(cty.Number),
			expectedVal: cty.NullVal(cty.Number),
		},
		{
			name: "integer_on_nonnumber_constraint",
			obj: &dataspec.AttrSpec{
				Name:        "test",
				Type:        cty.String,
				DefaultVal:  cty.StringVal("hello"),
				Constraints: constraint.Integer,
			},
			asserts: diagtest.Asserts{
				{
					diagtest.IsError,
					diagtest.SummaryContains("Integer constraint", "non-numeric"),
				},
			},
		},
		{
			name: "min_max_unsupported_type",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.Bool,
				DefaultVal:   cty.True,
				MinInclusive: cty.NumberIntVal(1),
				MaxInclusive: cty.NumberIntVal(2),
			},
			asserts: diagtest.Asserts{
				{
					diagtest.IsError,
					diagtest.SummaryContains("MinInclusive can't be specified"),
				},
				{
					diagtest.IsError,
					diagtest.SummaryContains("MaxInclusive can't be specified"),
				},
			},
		},
		{
			name: "min_max_unsupported_type_of_constraints",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.Number,
				DefaultVal:   cty.Zero,
				MinInclusive: cty.StringVal("1"),
				MaxInclusive: cty.StringVal("2"),
			},
			asserts: diagtest.Asserts{
				{
					diagtest.IsError,
					diagtest.SummaryContains("MinInclusive", "must be a number"),
				},
				{
					diagtest.IsError,
					diagtest.SummaryContains("MaxInclusive", "must be a number"),
				},
			},
		},
		{
			name: "min_max_unsupported_type_of_constraints_2",
			obj: &dataspec.AttrSpec{
				Name:         "test",
				Type:         cty.Number,
				DefaultVal:   cty.Zero,
				MinInclusive: cty.NullVal(cty.Number),
				MaxInclusive: cty.NullVal(cty.Number),
			},
			asserts: diagtest.Asserts{
				{
					diagtest.IsError,
					diagtest.SummaryContains("MinInclusive", "must be non-null"),
				},
				{
					diagtest.IsError,
					diagtest.SummaryContains("MaxInclusive", "must be non-null"),
				},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// t.Parallel()
			spec := dataspec.RootSpec{
				Attrs: []*dataspec.AttrSpec{
					tc.obj,
				},
			}
			diags := spec.ValidateSpec()
			if diags.HasErrors() {
				tc.asserts.AssertMatch(t, diags, nil)
				return
			}
			block := &hclsyntax.Block{
				Type: "anyname",
				Body: &hclsyntax.Body{
					Attributes: hclsyntax.Attributes{},
				},
			}
			if tc.inputVal != cty.NilVal {
				block.Body.Attributes[tc.obj.Name] = &hclsyntax.Attribute{
					Name: tc.obj.Name,
					Expr: &hclsyntax.LiteralValueExpr{
						Val: tc.inputVal,
					},
				}
			}

			objVal, diag := dataspec.DecodeAndEvalBlock(context.Background(), block, &spec, nil)
			diags.Extend(diag)
			tc.asserts.AssertMatch(t, diags, nil)
			if diags.HasErrors() {
				return
			}

			val := objVal.GetAttrVal(tc.obj.Name)
			if !val.RawEquals(tc.expectedVal) {
				t.Fatalf("Values not equal. Expected: %s; Got: %s", tc.expectedVal.GoString(), val.GoString())
			}
		})
	}
}
