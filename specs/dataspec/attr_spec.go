// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package dataspec

import (
	"bytes"
	_ "embed"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"strings"
	"text/template"
	"unicode/utf8"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

//go:embed attr_spec_doc_comment.gotmpl
var docTmplBody string

var attrSpecDocCommentTmpl = template.Must(template.New("attr_spec_doc_comment").Parse(docTmplBody))

type AttrSpec struct {
	Name       string
	Type       cty.Type
	DefaultVal cty.Value
	ExampleVal cty.Value
	Doc        string

	Constraints constraint.Constraints
	// If set then value must be one of the specified values
	OneOf constraint.OneOf
	// For numbers - min value; for collections - min number of elements; for strings - min length
	MinInclusive cty.Value
	// For numbers - max value; for collections - max number of elements; for strings - max length
	MaxInclusive cty.Value
	// If specified – a deprecation warning would appear if an attribute is specified
	Deprecated string
	// If set then the value is hidden in logs
	Secret bool
}

func (a *AttrSpec) computeMinInclusive() cty.Value {
	// we have constraint.NonEmpty constraint on a collection type
	if a.Constraints.Is(constraint.NonEmpty) && // has NonEmpty
		(!a.Type.IsPrimitiveType() || a.Type != cty.Number) && // is not a number
		(a.MinInclusive.IsNull() || // Has MinInclusive < 1 or not set
			a.MinInclusive.LessThan(cty.NumberIntVal(1)).True()) {
		return cty.NumberIntVal(1)
	}
	return a.MinInclusive
}

func formatType(buf *strings.Builder, t cty.Type) {
	if t.IsTupleType() {
		buf.WriteString("[")
		types := t.TupleElementTypes()
		if len(types) > 0 {
			formatType(buf, types[0])
			for _, ty := range types[1:] {
				buf.WriteString(", ")
				formatType(buf, ty)
			}
		}
		buf.WriteString("]")
	} else {
		buf.WriteString(t.FriendlyNameForConstraint())
	}
}

func (a *AttrSpec) DocComment() hclwrite.Tokens {
	isRequired := a.Constraints.Is(constraint.Required)

	var attrType string
	if a.Constraints.Is(constraint.Integer) {
		attrType = "integer"
	} else {
		var buf strings.Builder
		formatType(&buf, a.Type)
		attrType = buf.String()
	}

	var oneOf string
	if !a.OneOf.IsEmpty() {
		oneOf = a.OneOf.String()
	}
	min := a.computeMinInclusive()
	max := a.MaxInclusive

	var minVal string
	var maxVal string

	var minLenVal string
	var maxLenVal string

	if a.Type.IsPrimitiveType() && a.Type == cty.Number {
		if !min.IsNull() {
			minVal = min.AsBigFloat().String()
		}
		if !max.IsNull() {
			maxVal = max.AsBigFloat().String()
		}
	} else {
		if !min.IsNull() {
			minLen, _ := min.AsBigFloat().Uint64()
			minLenVal = strconv.FormatUint(minLen, 10)
		}
		if !max.IsNull() {
			maxLen, _ := max.AsBigFloat().Uint64()
			maxLenVal = strconv.FormatUint(maxLen, 10)
		}
	}

	var example string
	if !isRequired {
		if a.ExampleVal != cty.NilVal {
			f := hclwrite.NewEmptyFile()
			f.Body().SetAttributeValue(a.Name, a.ExampleVal)
			example = string(hclwrite.Format(f.Bytes()))
		}
	}

	trimmedDoc := strings.Trim(a.Doc, "\n ")

	details := map[string]any{
		"Doc":        trimmedDoc,
		"IsRequired": isRequired,
		"Type":       attrType,
		"OneOf":      oneOf,
		"MinVal":     minVal,
		"MaxVal":     maxVal,
		"MinLenVal":  minLenVal,
		"MaxLenVal":  maxLenVal,
		"Example":    example,
	}

	var docVal bytes.Buffer
	if err := attrSpecDocCommentTmpl.Execute(&docVal, details); err != nil {
		slog.Error("Error while rendering an attribute doc template", "err", err)
		panic("Error while rendering an attribute doc template")
	}

	return comment(nil, docVal.String())
}

func (a *AttrSpec) WriteDoc(w *hclwrite.Body) {
	// write out documentation
	w.AppendUnstructuredTokens(a.DocComment())

	// write the attribute
	var val cty.Value
	if a.Constraints.Is(constraint.Required) {
		val = a.ExampleVal
		if val.IsNull() {
			val = exampleValueForType(a.Type)
		}
	} else {
		val = a.DefaultVal
	}

	w.SetAttributeValue(a.Name, val)
}

func trimSpace(val cty.Value) cty.Value {
	if !val.IsNull() && val.Type().Equals(cty.String) && val.IsKnown() {
		var marks cty.ValueMarks
		val, marks = val.Unmark()
		val = cty.StringVal(strings.TrimSpace(val.AsString()))
		return val.WithMarks(marks)
	}
	return val
}

func ctyToInt(val cty.Value) int64 {
	i, acc := val.AsBigFloat().Int64()
	if acc != big.Exact {
		panic(fmt.Sprintf("%s is not an exact integer", val.GoString()))
	}
	return i
}

func (a *AttrSpec) ValidateValue(val cty.Value) (diags diagnostics.Diag) {
	if val.IsNull() {
		if a.Constraints.Is(constraint.NonNull) {
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Attribute must be non-null",
				Detail:   fmt.Sprintf("The attribute %q is null.", a.Name),
			})
		}
	} else {
		// Length checks:
		length := -1
		if val.Type().IsCollectionType() || val.Type().IsTupleType() {
			length = val.LengthInt()
		} else if val.Type().IsPrimitiveType() && val.Type() == cty.String {
			length = utf8.RuneCountInString(val.AsString())
		}
		min := a.computeMinInclusive()
		max := a.MaxInclusive
		if length != -1 {
			length := int64(length)

			// length-validating constraints

			if !min.IsNull() && !max.IsNull() {
				min := ctyToInt(min)
				max := ctyToInt(max)
				if min > length || length > max {
					if min == max {
						diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Attribute length is not in range",
							Detail:   fmt.Sprintf("The length of attribute %q must be exactly %d", a.Name, min),
						})
					} else {
						diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Attribute length is not in range",
							Detail:   fmt.Sprintf("The length of attribute %q must be in range [%d; %d] (inclusive)", a.Name, min, max),
						})
					}
				}
			} else if !min.IsNull() {
				min := ctyToInt(min)
				if length < min {
					if min == 1 {
						diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Attribute must be non-empty",
							Detail:   fmt.Sprintf("Attribute %q can't be empty", a.Name),
						})
					} else {
						diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Attribute length is not in range",
							Detail:   fmt.Sprintf("The length of attribute %q must be >= %d", a.Name, min),
						})
					}
				}
			} else if !max.IsNull() {
				max := ctyToInt(max)
				if length > max {
					diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Attribute length is not in range",
						Detail:   fmt.Sprintf("The length of attribute %q must be <= %d", a.Name, max),
					})
				}
			}
		} else if val.Type().IsPrimitiveType() && val.Type() == cty.Number {
			// Numeric checks:
			if a.Constraints.Is(constraint.Integer) {
				_, acc := val.AsBigFloat().Int64()
				if acc != big.Exact {
					diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Attribute must be an integer",
						Detail:   fmt.Sprintf("The attribute %q must be an integer", a.Name),
					})
				}
			}
			// Range checks:
			if !min.IsNull() && !max.IsNull() {
				if val.GreaterThanOrEqualTo(min).And(val.LessThanOrEqualTo(max)).False() {
					diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Attribute is not in range",
						Detail:   fmt.Sprintf("The attribute %q must be in range [%s; %s] (inclusive)", a.Name, min.AsBigFloat().String(), max.AsBigFloat().String()),
					})
				}
			} else if !min.IsNull() {
				if val.GreaterThanOrEqualTo(min).False() {
					diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Attribute is not in range",
						Detail:   fmt.Sprintf("The attribute %q must be >= %s", a.Name, min.AsBigFloat().String()),
					})
				}
			} else if !max.IsNull() {
				if val.LessThanOrEqualTo(max).False() {
					diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Attribute is not in range",
						Detail:   fmt.Sprintf("The attribute %q value must be <= %s", a.Name, max.AsBigFloat().String()),
					})
				}
			}
		}
	}
	if !a.OneOf.Validate(val) {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Attribute is not one of the allowed values",
			Detail:   fmt.Sprintf("The attribute %q must be one of: %s", a.Name, a.OneOf),
		})
	}
	return diags
}

func (a *AttrSpec) ValidateSpec() (diags diagnostics.Diag) {
	if a.Constraints.Is(constraint.Required) {
		if a.ExampleVal == cty.NilVal {
			diags.AddWarn(fmt.Sprintf("Missing example value for a required attribute %q", a.Name), "")
		}
		if a.DefaultVal != cty.NilVal {
			diags.Add(
				fmt.Sprintf(
					"Default value is specified for a required attribute %q = %s",
					a.Name,
					a.DefaultVal.GoString(),
				),
				"",
			)
		}
	}

	if a.Constraints.Is(constraint.Integer) && !a.Type.Equals(cty.Number) {
		diags.Add(fmt.Sprintf("Integer constraint is specified for a non-numeric attribute %q", a.Name), "")
	}
	min := a.MinInclusive

	max := a.MaxInclusive
	skipMinMaxRelativeCheck := false
	for _, v := range []struct {
		name string
		val  cty.Value
	}{{"MinInclusive", min}, {"MaxInclusive", max}} {
		if v.val == cty.NilVal {
			continue
		}

		if (a.Type.IsPrimitiveType() && a.Type == cty.Bool) || a.Type.IsCapsuleType() {
			diags.Add(fmt.Sprintf("%s can't be specified for %s %q", v.name, a.Type.FriendlyName(), a.Name), "")
			skipMinMaxRelativeCheck = true
			continue
		}
		if !v.val.Type().IsPrimitiveType() || v.val.Type() != cty.Number {
			diags.Add(
				fmt.Sprintf(
					"%s specified for %q must be a number, not %s",
					v.name,
					a.Name,
					v.val.Type().FriendlyName(),
				),
				"",
			)
			skipMinMaxRelativeCheck = true
			continue
		}
		if v.val.IsNull() {
			diags.Add(fmt.Sprintf("%s specified for %q must be non-null", v.name, a.Name), "")
			skipMinMaxRelativeCheck = true
			continue
		}

		if !a.Type.Equals(cty.Number) {
			// Min is length, must be an num >=0
			num, acc := v.val.AsBigFloat().Int64()
			if acc != big.Exact {
				diags.Add(fmt.Sprintf("%s specified for %q must be an integer", v.name, a.Name), "")
			}
			if num < 0 {
				diags.Add(fmt.Sprintf("%s specified for %q must be >= 0", v.name, a.Name), "")
			}
		}
	}

	if !skipMinMaxRelativeCheck && !min.IsNull() && !max.IsNull() {
		// no errors - values are numbers and can be compared
		if min.LessThanOrEqualTo(max).False() {
			diags.Add(fmt.Sprintf("%q: MinInclusive must be <= MaxInclusive", a.Name), "")
		}
	}

	if len(diags) == 0 {
		if a.DefaultVal != cty.NilVal {
			diag := a.ValidateValue(a.DefaultVal)
			prefix := fmt.Sprintf("Default value for attribute %q: ", a.Name)
			for _, d := range diag {
				if d.Severity != hcl.DiagError {
					continue
				}
				d.Summary = prefix + d.Summary
				diags.Append(d)
			}
		}
		if a.ExampleVal != cty.NilVal {
			diag := a.ValidateValue(a.ExampleVal)
			prefix := fmt.Sprintf("Example value for attribute %q: ", a.Name)
			for _, d := range diag {
				if d.Severity != hcl.DiagError {
					continue
				}
				d.Summary = prefix + d.Summary
				diags.Append(d)
			}
		}
	}
	return diags
}
