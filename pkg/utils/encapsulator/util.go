// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

// Package encapsulator provides type-safe conversions to and from cty.CapsuleValue.
package encapsulator

import (
	"github.com/zclconf/go-cty/cty"
)

// isValid tests that cty.Value is safe to work with.
func isValid(v cty.Value) bool {
	return !v.IsNull() && v.IsKnown()
}

// Compatible checks that values produced by the given Encoder are decodable by the given Decoder.
// This is a loose check, for example it allows decoding value that implements interface DT.
func Compatible(encoder EncoderI, decoder DecoderI) bool {
	return decoder.Decodable(encoder.CtyType())
}
