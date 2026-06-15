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

// Sets slice[idx] = val, growing the slice if needed, and returns the updated slice.
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

// Produce a new slice by applying function fn to items of the slice s.
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

// Produce a new slice by applying (possibly erroring) function fn to items of the slice s.
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

// Produce a new slice by applying function fn to items of the slice s.
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
