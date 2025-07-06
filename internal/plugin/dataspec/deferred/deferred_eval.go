package deferred

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/utils/encapsulator"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

type Evaluable interface {
	DeferredEval(ctx context.Context, dataCtx plugindata.Map) (result cty.Value, diags diagnostics.Diag)
}

var deferredEvalReflectType = reflect.TypeFor[Evaluable]()

// Type represents cty-type-erased deferred evaluation. This type is used to
// hide custom decoder on an inner type (i.e. JqQuery) to avoid repeated
// calls to custom decode.
var Type *encapsulator.Codec[deferredEval]

func init() {
	Type = encapsulator.NewCodec("Deferred Evaluation", &encapsulator.CapsuleOps[deferredEval]{
		ConversionTo: func(dst cty.Type) func(cty.Value, cty.Path) (*deferredEval, error) {
			if !reflect.PointerTo(dst.EncapsulatedType()).Implements(deferredEvalReflectType) {
				return nil
			}
			return func(v cty.Value, p cty.Path) (*deferredEval, error) {
				if v.IsNull() {
					return nil, p.NewErrorf("can't convert null to DeferredEval")
				}
				if !v.IsKnown() {
					return nil, p.NewErrorf("can't convert unknown value to DeferredEval")
				}
				val := v.EncapsulatedValue()
				if val == nil {
					return nil, p.NewErrorf("can't convert nil value to DeferredEval")
				}
				if defEvalVal, ok := val.(*deferredEval); ok {
					// Avoid double wrapping (shouldn't happen in practice)
					return defEvalVal, nil
				}
				evaluable := val.(Evaluable) //nolint // we know it's Evaluable
				return &deferredEval{
					Evaluable: evaluable,
				}, nil
			}
		},
		ConversionFrom: func(src cty.Type) func(*deferredEval, cty.Path) (cty.Value, error) {
			if Type.CtyTypeEqual(src) {
				// Conversion to self is a no-op
				return func(de *deferredEval, _ cty.Path) (cty.Value, error) {
					return Type.ToCty(de), nil
				}
			}
			slog.Error("Conversion of DeferredEval type is prohibited, evaluate it instead")
			return nil
		},
	})
}

type deferredEval struct {
	Evaluable
}
