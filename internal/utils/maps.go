package utils

import "github.com/blackstork-io/fabric/internal/utils/diagnostics"

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

// If key in map - return corresponding value and delete it from map
func Pop[K comparable, V any](m map[K]V, key K) (val V, found bool) {
	if m == nil {
		return
	}
	val, found = m[key]
	if found {
		delete(m, key)
	}
	return
}

// Produce a new map by applying function fn to the values of the map m.
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

// Produce a new map by applying (possibly erroring) function fn to the values of the map m.
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

// Produce a new map by applying function fn to items of the map m.
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
