// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package encapsulator

import (
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/customdecode"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
)

type typedOpsI interface {
	asCtyCapsuleOps(encoder *encoderCore) (res *cty.CapsuleOps)
}

// CapsuleOps represents a set of overloaded operations for a capsule type.
// This struct is identical to cty.CapsuleOps, except that it specifies
// the types of the arguments to functions. It will be converted to
// cty.CapsuleOps behind the scenes.
//
// Each field is a reference to a function that can either be nil or can be
// set to an implementation of the corresponding operation. If an operation
// function is nil then it isn't supported for the given capsule type.
type CapsuleOps[T any] struct {
	// GoString provides the GoString implementation for values of the
	// corresponding type. Conventionally this should return a string
	// representation of an expression that would produce an equivalent
	// value.
	GoString func(val *T) string

	// TypeGoString provides the GoString implementation for the corresponding
	// capsule type itself.
	TypeGoString func(goTy reflect.Type) string

	// Equals provides the implementation of the Equals operation. This is
	// called only with known, non-null values of the corresponding type,
	// but if the corresponding type is a compound type then it must be
	// ready to detect and handle nested unknown or null values, usually
	// by recursively calling Value.Equals on those nested values.
	//
	// The result value must always be of type cty.Bool, or the Equals
	// operation will panic.
	//
	// If RawEquals is set without also setting Equals, the RawEquals
	// implementation will be used as a fallback implementation. That fallback
	// is appropriate only for leaf types that do not contain any nested
	// cty.Value that would need to distinguish Equals vs. RawEquals for their
	// own equality.
	//
	// If RawEquals is nil then Equals must also be nil, selecting the default
	// pointer-identity comparison instead.
	Equals func(a, b *T) cty.Value

	// RawEquals provides the implementation of the RawEquals operation.
	// This is called only with known, non-null values of the corresponding
	// type, but if the corresponding type is a compound type then it must be
	// ready to detect and handle nested unknown or null values, usually
	// by recursively calling Value.RawEquals on those nested values.
	//
	// If RawEquals is nil, values of the corresponding type are compared by
	// pointer identity of the encapsulated value.
	RawEquals func(a, b *T) bool

	// HashKey provides a hashing function for values of the corresponding
	// capsule type. If defined, cty will use the resulting hashes as part
	// of the implementation of sets whose element type is or contains the
	// corresponding capsule type.
	//
	// If a capsule type defines HashValue then the function _must_ return
	// an equal hash value for any two values that would cause Equals or
	// RawEquals to return true when given those values. If a given type
	// does not uphold that assumption then sets including this type will
	// not behave correctly.
	HashKey func(v *T) string

	// ConversionFrom can provide conversions from the corresponding type to
	// some other type when values of the corresponding type are used with
	// the "convert" package. (The main cty package does not use this operation.)
	//
	// This function itself returns a function, allowing it to switch its
	// behavior depending on the given source type. Return nil to indicate
	// that no such conversion is available.
	ConversionFrom func(src cty.Type) func(*T, cty.Path) (cty.Value, error)

	// ConversionTo can provide conversions to the corresponding type from
	// some other type when values of the corresponding type are used with
	// the "convert" package. (The main cty package does not use this operation.)
	//
	// This function itself returns a function, allowing it to switch its
	// behavior depending on the given destination type. Return nil to indicate
	// that no such conversion is available.
	ConversionTo func(dst cty.Type) func(cty.Value, cty.Path) (*T, error)

	// ExtensionData is an extension point for applications that wish to
	// create their own extension features using capsule types.
	//
	// The key argument is any value that can be compared with Go's ==
	// operator, but should be of a named type in a package belonging to the
	// application defining the key. An ExtensionData implementation must
	// check to see if the given key is familiar to it, and if so return a
	// suitable value for the key.
	//
	// If the given key is unrecognized, the ExtensionData function must
	// return a nil interface. (Importantly, not an interface containing a nil
	// pointer of some other type.)
	// The common implementation of ExtensionData is a single switch statement
	// over "key" which has a default case returning nil.
	//
	// The meaning of any given key is entirely up to the application that
	// defines it. Applications consuming ExtensionData from capsule types
	// should do so defensively: if the result of ExtensionData is not valid,
	// prefer to ignore it or gracefully produce an error rather than causing
	// a panic.
	ExtensionData func(key any) any

	// CustomExpressionDecoder is a function that overrides the usual
	// hcl evaluation. It takes precedence over the function that may
	// be returned from ExtensionData
	CustomExpressionDecoder func(expr hcl.Expression, evalCtx *hcl.EvalContext) (*T, diagnostics.Diag)
}

func (co *CapsuleOps[T]) asCtyCapsuleOps(encoder *encoderCore) (res *cty.CapsuleOps) {
	if co == nil {
		return res
	}
	res = &cty.CapsuleOps{
		TypeGoString: co.TypeGoString,
	}

	if co.ExtensionData != nil || co.CustomExpressionDecoder != nil {
		res.ExtensionData = func(key any) any {
			switch key {
			case customdecode.CustomExpressionDecoder:
				if co.CustomExpressionDecoder == nil {
					break
				}
				return customdecode.CustomExpressionDecoderFunc(func(expr hcl.Expression, ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
					data, diags := co.CustomExpressionDecoder(expr, ctx)
					diag := hcl.Diagnostics(diags)
					if data == nil && diag.HasErrors() {
						return cty.NilVal, diag
					}
					return encoder.toCty(data), diag
				})
			}
			if co.ExtensionData != nil {
				return co.ExtensionData(key)
			}
			return nil
		}
	}
	if co.GoString != nil {
		res.GoString = func(val any) string {
			return co.GoString(val.(*T))
		}
	}
	if co.Equals != nil {
		res.Equals = func(a, b any) cty.Value {
			return co.Equals(a.(*T), b.(*T))
		}
	}
	if co.RawEquals != nil {
		res.RawEquals = func(a, b any) bool {
			return co.RawEquals(a.(*T), b.(*T))
		}
	}
	if co.HashKey != nil {
		res.HashKey = func(v any) string {
			return co.HashKey(v.(*T))
		}
	}
	if co.ConversionFrom != nil {
		res.ConversionFrom = func(src cty.Type) func(val any, path cty.Path) (cty.Value, error) {
			fn := co.ConversionFrom(src)
			if fn == nil {
				return nil
			}
			return func(val any, path cty.Path) (cty.Value, error) {
				return fn(val.(*T), path)
			}
		}
	}
	if co.ConversionTo != nil {
		res.ConversionTo = func(dst cty.Type) func(val cty.Value, path cty.Path) (any, error) {
			fn := co.ConversionTo(dst)
			if fn == nil {
				return nil
			}
			return func(val cty.Value, path cty.Path) (any, error) {
				return fn(val, path)
			}
		}
	}
	return res
}
