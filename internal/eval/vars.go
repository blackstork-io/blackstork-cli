package eval

import (
	"context"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

// Evaluates `variables` and stores the results in `dataCtx` under the key "vars".
func ApplyVars(ctx context.Context, variables *definitions.Vars, dataCtx plugindata.Map) (diags diagnostics.Diag) {
	if variables.Empty() {
		return
	}
	varsData := dataCtx[definitions.BlockKindVars]

	var vars plugindata.Map
	if varsData == nil {
		vars = plugindata.Map{}
	} else {
		vars = varsData.(plugindata.Map)
	}
	var diag diagnostics.Diag
	for _, attr := range variables.GetAttrs() {
		vars[attr.Name], diag = evalVar(ctx, dataCtx, attr)
		diags.Extend(diag.Refine(
			diagnostics.DefaultSubject(attr.ValueRange),
		))
	}
	dataCtx["vars"] = vars
	return
}

func evalVar(
	ctx context.Context,
	dataCtx plugindata.Map,
	attr *dataspec.Attr,
) (data plugindata.Data, diags diagnostics.Diag) {
	val, diags := dataspec.EvalAttr(ctx, attr, dataCtx)
	if diags.HasErrors() {
		return
	}
	dataVal, err := plugindata.Encapsulated.FromCty(val)
	if diags.AppendErr(err, "Failed to convert a variable value") {
		return
	}
	if dataVal != nil {
		data = *dataVal
	}
	return
}

func verifyRequiredVars(docDataCtx plugindata.Map, requiredVars []string) (diag diagnostics.Diag) {
	vars, varsPresent := docDataCtx["vars"].(plugindata.Map)
	for _, reqVar := range requiredVars {
		if !varsPresent || vars[reqVar] == nil {
			return diagnostics.FromHcl(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing a required variable",
				Detail:   "The block requires `" + reqVar + "` var which is not set.",
			})
		}
	}
	return nil
}
