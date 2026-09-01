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

func Contains[K comparable, V any](m map[K]V, key K) bool {
	_, found := m[key]
	return found
}

func SliceToSet[K comparable](slice []K) map[K]struct{} {
	res := make(map[K]struct{}, len(slice))
	for _, v := range slice {
		res[v] = struct{}{}
	}
	return res
}

// Pop returns and deletes the value for a key.
func Pop[K comparable, V any](m map[K]V, key K) (val V, found bool) {
	if m == nil {
		return val, found
	}
	val, found = m[key]
	if found {
		delete(m, key)
	}
	return val, found
}

// MapMap returns a map produced by applying fn to each value in m.
func MapMap[K comparable, VIn, VOut any](m map[K]VIn, fn func(VIn) VOut) map[K]VOut {
	if m == nil {
		return nil
	}
	out := make(map[K]VOut, len(m))
	for k, v := range m {
		out[k] = fn(v)
	}
	return out
}

// MapMapErr returns a map produced by applying a possibly failing function to each value.
// Returns on the first error with nil map.
func MapMapErr[K comparable, VIn, VOut any](m map[K]VIn, fn func(VIn) (VOut, error)) (map[K]VOut, error) {
	var err error
	if m == nil {
		return nil, nil
	}
	out := make(map[K]VOut, len(m))
	for k, v := range m {
		out[k], err = fn(v)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// MapMapDiags returns a map produced by applying a diagnostic-producing function to each item.
// Collects returned diagnostics
func MapMapDiags[K comparable, VIn, VOut any](diags *diagnostics.Diag, m map[K]VIn, fn func(VIn) (VOut, diagnostics.Diag)) map[K]VOut {
	if m == nil {
		return nil
	}
	var diag diagnostics.Diag
	out := make(map[K]VOut, len(m))
	for k, v := range m {
		out[k], diag = fn(v)
		diags.Extend(diag)
	}
	return out
}
