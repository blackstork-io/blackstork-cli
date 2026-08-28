// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package deferred

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/itchyny/gojq"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/pkg/utils/encapsulator"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
)

type JqQuery struct {
	query     string
	srcRange  *hcl.Range
	parseOnce func() (*gojq.Code, diagnostics.Diag)
}

var JqQueryType = encapsulator.NewCodec("jq query", &encapsulator.CapsuleOps[JqQuery]{
	CustomExpressionDecoder: func(expr hcl.Expression, evalCtx *hcl.EvalContext) (val *JqQuery, diags diagnostics.Diag) {
		queryVal, diag := expr.Value(evalCtx)
		if diags.Extend(diag) {
			return val, diags
		}
		if queryVal.IsNull() || !queryVal.IsKnown() || !queryVal.Type().Equals(cty.String) {
			diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Invalid argument",
				Detail:      "A string is required",
				Subject:     expr.Range().Ptr(),
				Expression:  expr,
				EvalContext: evalCtx,
			})
			return val, diags
		}

		val = &JqQuery{
			query:    queryVal.AsString(),
			srcRange: expr.Range().Ptr(),
		}
		val.parseOnce = utils.OnceVal(val.parse)
		return val, diags
	},
})

const funcName = "query_jq"

// WithQueryFuncs adds "query_jq" function to the eval context
func WithQueryFuncs(evalCtx *hcl.EvalContext) *hcl.EvalContext {
	// try finding existing jq eval context
	if jqEvalCtx := utils.EvalContextByFunc(evalCtx, funcName); jqEvalCtx != nil {
		return evalCtx
	}

	evalCtx = evalCtx.NewChild()
	evalCtx.Functions = map[string]function.Function{
		funcName: function.New(&function.Spec{
			Params: []function.Parameter{
				{
					Name:        "query",
					Description: "The jq query string",
					Type:        JqQueryType.CtyType(),
				},
			},
			Type: function.StaticReturnType(Type.CtyType()),
			Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
				return convert.Convert(args[0], retType)
			},
		}),
	}
	return evalCtx
}

func (q *JqQuery) parse() (code *gojq.Code, diags diagnostics.Diag) {
	if q == nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse the query",
			Detail:   "Query wasn't defined",
		})
		return code, diags
	}
	jqQuery, err := gojq.Parse(q.query)
	if err != nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse the query",
			Detail:   err.Error(),
			Subject:  q.srcRange,
			Extra: diagnostics.GoJQError{
				Err:   err,
				Query: q.query,
			},
		})
		return code, diags
	}

	code, err = gojq.Compile(jqQuery)
	if err != nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to compile the query",
			Detail:   err.Error(),
			Subject:  q.srcRange,
			Extra: diagnostics.GoJQError{
				Err:   err,
				Query: q.query,
			},
		})
	}
	return code, diags
}

func (q *JqQuery) DeferredEval(
	ctx context.Context,
	dataCtx plugindata.Map,
) (_ cty.Value, diags diagnostics.Diag) {
	if q == nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to evaluate the query",
			Detail:   "Query wasn't defined",
		})
		return
	}
	defer func() {
		diags.Refine(diagnostics.DefaultSubject(*q.srcRange))
	}()
	if dataCtx == nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to evaluate the query",
			Detail:   "Data context wasn't provided. Make sure the query is used in the content provider invocation.",
		})
		return
	}
	code, diags := q.parseOnce()
	if diags.HasErrors() {
		return
	}
	res, hasResult := code.RunWithContext(ctx, dataCtx.Any()).Next()
	err, ok := res.(error)
	if ok {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to run the query",
			Detail:   err.Error(),
			Extra: diagnostics.GoJQError{
				Err:   err,
				Query: q.query,
			},
		})
		return
	}
	if !hasResult {
		res = nil
	}
	data, err := plugindata.ParseAny(res)
	if err != nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incorrect query result type",
			Detail:   err.Error(),
		})
		return
	}
	return plugindata.Encapsulated.ToCty(&data), diags
}
