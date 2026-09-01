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

	"github.com/zclconf/go-cty/cty"
)

// EncoderI is the non-generic encoder interface.
type EncoderI interface {
	CtyType() cty.Type
	CtyTypeEqual(ty cty.Type) bool
	ValCtyTypeEqual(v cty.Value) bool
	toCty(val any) cty.Value
}

type encoderCore struct {
	ctyType cty.Type
}

func (e *encoderCore) toCty(val any) cty.Value {
	return cty.CapsuleVal(e.ctyType, val)
}

// CtyType returns the cty type of the encoded values.
func (e *encoderCore) CtyType() cty.Type {
	return e.ctyType
}

// CtyTypeEqual checks if the given cty type can be decoded into T.
// This is the strictest check, requiring that the given cty type was produced by this Encoder.
func (e *encoderCore) CtyTypeEqual(ty cty.Type) bool {
	return e.ctyType.Equals(ty)
}

// ValCtyTypeEqual checks if the given cty value can be decoded into T.
// This is the strictest check, requiring that the given cty value was produced by this Encoder.
func (e *encoderCore) ValCtyTypeEqual(v cty.Value) bool {
	v, _ = v.Unmark()
	return isValid(v) && e.CtyTypeEqual(v.Type())
}

func (e *encoderCore) initEncoderCore(name string, nativeType reflect.Type, ops typedOpsI) {
	ctyOps := ops.asCtyCapsuleOps(e)
	if ctyOps == nil {
		e.ctyType = cty.Capsule(name, nativeType)
	} else {
		e.ctyType = cty.CapsuleWithOps(name, nativeType, ctyOps)
	}
}

// Encoder encapsulates *T into cty.Value.
type Encoder[T any] struct {
	encoderCore
}

// ToCty encapsulates *T into cty.Value.
func (e *Encoder[T]) ToCty(val *T) cty.Value {
	return e.toCty(val)
}

// ValToCty is a convenience function that encapsulates &val into cty.Value.
func (e *Encoder[T]) ValToCty(val T) cty.Value {
	return e.toCty(&val)
}

// NewEncoder creates an Encoder.
// Will use cty.Capsule if capsuleOps is nil, and cty.CapsuleWithOps otherwise
func NewEncoder[T any](friendlyName string, capsuleOps *CapsuleOps[T]) *Encoder[T] {
	enc := &Encoder[T]{}
	enc.initEncoderCore(friendlyName, reflect.TypeFor[T](), capsuleOps)
	return enc
}
