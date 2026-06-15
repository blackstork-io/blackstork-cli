// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package encapsulator_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/blackstork-io/blackstork-cli/pkg/utils/encapsulator"
)

func TestTypedCapsuleOpts(t *testing.T) {
	assert := assert.New(t)
	i := 1337
	origI := &i

	var c *encapsulator.Codec[int]
	var calls []string

	c = encapsulator.NewCodec[int](
		"test int",
		&encapsulator.CapsuleOps[int]{
			GoString: func(i *int) string {
				assert.Same(origI, i)
				calls = append(calls, "GoString")
				return "GoString"
			},
			TypeGoString: func(goTy reflect.Type) string {
				assert.Equal(reflect.TypeFor[int](), goTy)
				calls = append(calls, "TypeGoString")
				return "TypeGoString"
			},
			Equals: func(i1, i2 *int) cty.Value {
				assert.Same(origI, i1)
				assert.Same(origI, i2)
				calls = append(calls, "Equals")
				return cty.BoolVal(true)
			},
			RawEquals: func(i1, i2 *int) bool {
				assert.Same(origI, i1)
				assert.Same(origI, i2)
				calls = append(calls, "RawEquals")
				return true
			},
			HashKey: func(v *int) string {
				assert.Same(origI, v)
				calls = append(calls, "HashKey")
				return "HashKey"
			},
			ConversionFrom: func(src cty.Type) func(*int, cty.Path) (cty.Value, error) {
				assert.True(src.Equals(cty.String))
				calls = append(calls, "ConversionFrom")
				return func(i *int, p cty.Path) (cty.Value, error) {
					assert.Same(origI, i)
					calls = append(calls, "ConversionFromFn")
					return cty.StringVal("ConversionFrom"), nil
				}
			},
			ConversionTo: func(dst cty.Type) func(cty.Value, cty.Path) (*int, error) {
				assert.True(dst.Equals(cty.String))
				calls = append(calls, "ConversionTo")
				return func(v cty.Value, p cty.Path) (*int, error) {
					assert.Equal("ConversionTo", v.AsString())
					calls = append(calls, "ConversionToFn")
					return origI, nil
				}
			},
		},
	)

	v := c.ToCty(origI)
	assert.Equal("GoString", v.GoString())
	assert.Equal("TypeGoString", v.Type().GoString())
	assert.True(v.Equals(v).True())
	assert.True(v.RawEquals(v))
	assert.NotPanics(func() {
		v.Hash()
	})
	val, err := convert.Convert(v, cty.String)
	assert.NoError(err)
	assert.Equal("ConversionFrom", val.AsString())
	val, err = convert.Convert(cty.StringVal("ConversionTo"), c.CtyType())
	assert.NoError(err)
	assert.Same(origI, c.MustFromCty(val))
	assert.Equal(
		[]string{
			"GoString",
			"TypeGoString",
			"Equals",
			"RawEquals",
			"HashKey",
			"ConversionFrom",
			"ConversionFromFn",
			"ConversionTo",
			"ConversionToFn",
		},
		calls,
	)
}

func TestTypedCapsuleOptsNonConvertible(t *testing.T) {
	assert := assert.New(t)
	i := 1337
	origI := &i

	var c *encapsulator.Codec[int]
	var calls []string

	c = encapsulator.NewCodec[int](
		"test int",
		&encapsulator.CapsuleOps[int]{
			ConversionFrom: func(src cty.Type) func(*int, cty.Path) (cty.Value, error) {
				assert.True(src.Equals(cty.String))
				calls = append(calls, "ConversionFrom")
				return nil
			},
			ConversionTo: func(dst cty.Type) func(cty.Value, cty.Path) (*int, error) {
				assert.True(dst.Equals(cty.String))
				calls = append(calls, "ConversionTo")
				return nil
			},
		},
	)

	v := c.ToCty(origI)
	_, err := convert.Convert(v, cty.String)
	assert.NotNil(err)
	_, err = convert.Convert(cty.StringVal("ConversionTo"), c.CtyType())
	assert.NotNil(err)
	assert.Equal(
		[]string{
			"ConversionFrom",
			"ConversionTo",
		},
		calls,
	)
}
