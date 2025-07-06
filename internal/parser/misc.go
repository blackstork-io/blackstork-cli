package parser

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
)

type Ctyable interface {
	CtyType() cty.Type
}

// func ResolveWithDefined[B definitions.BlockDef](db *DefinedBlocks, expr hcl.Expression) (B, diagnostics.Diag) {
// 	blockMap := db.AsValueMap()
// 	return Resolve[B](blockMap, expr)
// }
// 
// func Resolve[B Ctyable](blockMap map[string]cty.Value, expr hcl.Expression) (B, diagnostics.Diag) {
// 
// 	var res B
// 
// 	val, diag := expr.Value(&hcl.EvalContext{
// 		Variables: blockMap,
// 	})
// 	var diags diagnostics.Diag
// 	if diags.Extend(diag) {
// 		return res, diags
// 	}
// 	expectedType := res.CtyType()
// 
// 	ty := val.Type()
// 	if !ty.Equals(expectedType) {
// 		diags.Append(&hcl.Diagnostic{
// 			Severity: hcl.DiagError,
// 			Summary:  "Incorrect reference",
// 			Detail: fmt.Sprintf(
// 				"Expected reference to `%s` but got a reference to `%s`",
// 				expectedType.FriendlyName(),
// 				ty.FriendlyName(),
// 			),
// 			Subject: expr.Range().Ptr(),
// 		})
// 		return res, diags
// 	}
// 	res = val.EncapsulatedValue().(B)
// 	return res, diags
// }

//res = .(B) //nolint:forcetypeassert // This type assertion is done via cty in db.resolve

func makeNamePlaceholder() string {
	id := uuid.New()
	return id.String()[:8]
}
