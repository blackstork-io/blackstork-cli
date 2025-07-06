package utils

import (
	"reflect"
)

// Correct version of nil check, works on nil interfaces as well as any other value.
func IsNil(val any) bool {
	// Checking for nil on interface objects is terrible
	// Thanks to: https://stackoverflow.com/a/76595928/4632951
	if val == nil {
		return true
	}
	v := reflect.ValueOf(val)
	k := v.Kind()
	switch k {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer,
		reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	}

	return false
}

// Unwraps the value, panics if err is not nil.
func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

// Clone performs a shallow clone operation on pointer value
func Clone[T any](val *T) *T {
	if val == nil {
		return nil
	}
	valC := *val
	return &valC
}
