package utils

import "github.com/blackstork-io/fabric/internal/utils/diagnostics"

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
	return
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
