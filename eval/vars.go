// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package eval

import (
	"context"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

// ApplyVars evaluates variables and stores the results in `dataCtx` under the key "vars".
func ApplyVars(ctx context.Context, variables *definitions.Vars, dataCtx plugindata.Map) (diags diagnostics.Diag) {
	if variables.Empty() {
		return diags
	}
	varsData := dataCtx[definitions.BlockKindVars]

	var vars plugindata.Map
	if varsData == nil {
		vars = plugindata.Map{}
	} else {
		vars = varsData.(plugindata.Map)
	}
	// Make sure `dataCtx` has live list of vars updated after every evaluation
	dataCtx["vars"] = vars

	var diag diagnostics.Diag
	for _, attr := range variables.GetAttrs() {
		vars[attr.Name], diag = evalVar(ctx, dataCtx, attr)

		diags.Extend(diag.Refine(
			diagnostics.DefaultSubject(attr.ValueRange),
		))
	}
	return diags
}

func evalVar(
	ctx context.Context,
	dataCtx plugindata.Map,
	attr *dataspec.Attr,
) (data plugindata.Data, diags diagnostics.Diag) {
	val, diags := dataspec.EvalAttr(ctx, attr, dataCtx)
	if diags.HasErrors() {
		return data, diags
	}
	dataVal, err := plugindata.Encapsulated.FromCty(val)
	if diags.AppendErr(err, "Failed to convert a variable value") {
		return data, diags
	}
	if dataVal != nil {
		data = *dataVal
	}
	return data, diags
}

func verifyRequiredVars(docDataCtx plugindata.Map, requiredVars []string) (diag diagnostics.Diag) {
	vars, varsPresent := docDataCtx["vars"].(plugindata.Map)
	for _, reqVar := range requiredVars {
		if !varsPresent || vars[reqVar] == nil {
			return diagnostics.FromHCL(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing a required variable",
				Detail:   "The block requires `" + reqVar + "` var which is not set.",
			})
		}
	}
	return nil
}
