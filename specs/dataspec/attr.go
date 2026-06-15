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
	"fmt"
	"math/big"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

type Attr struct {
	Name       string
	NameRange  hcl.Range
	Value      cty.Value
	ValueRange hcl.Range
	Secret     bool
	// Spec that was used to decode this attribute.
	// This is only set if the block was decoded from hcl with a spec present.
	spec *AttrSpec
}

func (a *Attr) Range() hcl.Range {
	return hcl.RangeBetween(a.NameRange, a.ValueRange)
}

func (a *Attr) GetInt() (val int, diags diagnostics.Diag) {
	if a == nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid attribute",
			Detail:   "Attribute not found",
		})
		return val, diags
	}
	if !a.Value.IsKnown() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid attribute",
			Detail:   "Attribute value is unknown",
			Subject:  &a.ValueRange,
		})
		return val, diags
	}
	if a.Value.IsNull() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid attribute",
			Detail:   "Attribute value is null",
			Subject:  &a.ValueRange,
		})
		return val, diags
	}
	if !a.Value.Type().Equals(cty.Number) {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid attribute",
			Detail:   "Attribute is not a number",
			Subject:  &a.ValueRange,
		})
		return val, diags
	}
	v := a.Value.AsBigFloat()

	if !v.IsInt() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid attribute",
			Detail:   fmt.Sprintf("Attribute is not an integer (%s)", formatBigFloat(v, a.Secret)),
			Subject:  &a.ValueRange,
		})
		return val, diags
	}
	if v.Cmp(constraint.MaxInt) > 0 || v.Cmp(constraint.MinInt) < 0 {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid attribute",
			Detail:   fmt.Sprintf("Attribute value out of integer type range (%s)", formatBigFloat(v, a.Secret)),
			Subject:  &a.ValueRange,
		})
		return val, diags
	}
	val64, _ := v.Int64()
	return int(val64), diags
}

func (a *Attr) GetStringList() (_ []string, diags diagnostics.Diag) {
	if a == nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid attribute",
			Detail:   "Attribute not found",
		})
		return
	}
	if !a.Value.IsKnown() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid attribute",
			Detail:   "Attribute value is unknown",
			Subject:  &a.ValueRange,
		})
		return
	}
	if a.Value.IsNull() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid attribute",
			Detail:   "Attribute value is null",
			Subject:  &a.ValueRange,
		})
		return
	}
	val, err := convert.Convert(a.Value, cty.List(cty.String))
	if err != nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid attribute",
			Detail:   fmt.Sprintf("Attribute is not a list of strings (%s)", a.Value.Type().FriendlyName()),
			Subject:  &a.ValueRange,
		})
		return
	}
	res := make([]string, 0, val.LengthInt())
	for it := val.ElementIterator(); it.Next(); {
		_, v := it.Element()
		res = append(res, v.AsString())
	}

	return res, diags
}

func formatBigFloat(v *big.Float, isSecret bool) string {
	if isSecret {
		return "<secret value redacted>"
	}
	if v.IsInt() {
		return v.Text('f', 0)
	}
	return v.Text('f', -1)
}
