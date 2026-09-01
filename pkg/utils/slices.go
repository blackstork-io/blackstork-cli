// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package utils

import "github.com/blackstork-io/blackstork-cli/pkg/diagnostics"

// SetAt sets slice[idx] to val, growing the slice when needed.
func SetAt[T any](slice []T, idx int, val T) []T {
	needToAlloc := idx - len(slice)
	switch {
	case needToAlloc > 0:
		slice = append(slice, make([]T, needToAlloc)...)

		fallthrough
	case needToAlloc == 0:
		slice = append(slice, val)
	default:
		slice[idx] = val
	}
	return slice
}

// FnMap returns a slice produced by applying fn to each item in s.
func FnMap[I, O any](s []I, fn func(I) O) []O {
	if s == nil {
		return nil
	}
	out := make([]O, len(s))
	for i, v := range s {
		out[i] = fn(v)
	}
	return out
}

// FnMapErr returns a slice produced by applying a possibly failing function to each item.
// Returns on the first error with nil slice.
func FnMapErr[I, O any](s []I, fn func(I) (O, error)) (out []O, err error) {
	if s == nil {
		return nil, nil
	}
	out = make([]O, len(s))
	for i, v := range s {
		out[i], err = fn(v)
		if err != nil {
			out = nil
			break
		}
	}
	return out, err
}

// FnMapDiags returns a slice produced by applying a diagnostic-producing function to each item.
// Collects slice-like errors from the second return value (diagnostics in our case)
func FnMapDiags[I, O any](diags *diagnostics.Diag, s []I, fn func(I) (O, diagnostics.Diag)) []O {
	if s == nil {
		return nil
	}
	var diag diagnostics.Diag
	out := make([]O, len(s))
	for i, v := range s {
		out[i], diag = fn(v)
		diags.Extend(diag)
	}
	return out
}
