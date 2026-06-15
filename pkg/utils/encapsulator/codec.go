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
)

// Codec is both an Encoder and a Decoder for *T.
type Codec[T any] struct {
	Decoder[*T]
	Encoder[T]
}

// NewCodec creates a Codec (encoder + decoder).
// Will use cty.Capsule if capsuleOps is nil, and cty.CapsuleWithOps otherwise
func NewCodec[T any](friendlyName string, capsuleOps *CapsuleOps[T]) *Codec[T] {
	goType := reflect.TypeFor[T]()
	codec := &Codec[T]{
		Encoder: Encoder[T]{},
		Decoder: Decoder[*T]{
			decoderCore: decoderCore{
				goType: reflect.PointerTo(goType),
			},
		},
	}
	codec.initEncoderCore(friendlyName, goType, capsuleOps)
	return codec
}
