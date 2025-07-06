package encapsulator_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/encapsulator"
)

const testVal = int(1337)

func TestCodec(t *testing.T) {
	assert := assert.New(t)
	c := encapsulator.NewCodec[int]("test int", nil)
	i := testVal

	v := c.ToCty(&i)

	assert.True(encapsulator.Compatible(c, c))

	assert.True(c.ValDecodable(v))
	assert.True(c.ValTypeEqual(v))
	assert.True(c.ValCtyTypeEqual(v))

	ii, err := c.FromCty(v)
	assert.Equal(reflect.TypeOf(ii), c.GoType())

	assert.NoError(err)
	assert.Same(&i, ii)
	ii = c.MustFromCty(v)
	assert.Same(&i, ii)
}

func TestSeparateEncoderDecoder(t *testing.T) {
	assert := assert.New(t)
	e := encapsulator.NewEncoder[int]("test int", nil)
	i := testVal
	v := e.ToCty(&i)

	d := encapsulator.NewDecoder[*int]()

	assert.True(d.ValDecodable(v))
	assert.True(d.ValTypeEqual(v))

	ii, err := d.FromCty(v)
	assert.NoError(err)
	assert.Same(&i, ii)
	ii = d.MustFromCty(v)
	assert.Same(&i, ii)
}

func TestSeparateEncoderCodec(t *testing.T) {
	assert := assert.New(t)
	e := encapsulator.NewEncoder[int]("test int", nil)
	i := testVal
	v := e.ToCty(&i)

	c := encapsulator.NewCodec[int]("other int", nil)

	assert.True(c.ValDecodable(v))
	assert.True(c.ValTypeEqual(v))
	assert.False(c.ValCtyTypeEqual(v))

	ii, err := c.FromCty(v)
	assert.NoError(err)
	assert.Same(&i, ii)
	ii = c.MustFromCty(v)
	assert.Same(&i, ii)
}

func TestValToCty(t *testing.T) {
	assert := assert.New(t)
	c := encapsulator.NewCodec[int]("test int", nil)
	v := c.ValToCty(testVal)

	decoded, err := c.FromCty(v)
	assert.NoError(err)

	assert.Equal(testVal, *decoded)
}

type myType struct{}

type myInterface interface {
	myMethod()
}

func (*myType) myMethod() {
}

func TestImplicitCasting(t *testing.T) {
	assert := assert.New(t)
	e := encapsulator.NewEncoder[myType]("test int", nil)
	val := &myType{}
	v := e.ToCty(val)

	d := encapsulator.NewDecoder[myInterface]()

	assert.True(encapsulator.Compatible(e, d))

	assert.True(d.ValDecodable(v))
	assert.False(d.ValTypeEqual(v))

	decoded, err := d.FromCty(v)
	assert.NoError(err)
	assert.NotNil(decoded)
	assert.EqualValues(val, decoded)
	assert.Same(val, decoded.(*myType))
}

func TestDecoderErrors(t *testing.T) {
	assert := assert.New(t)

	d := encapsulator.NewDecoder[*int]()
	e := encapsulator.NewEncoder[int]("test int", nil)

	i, err := d.FromCty(cty.NullVal(e.CtyType()))
	assert.Nil(i)
	assert.ErrorIs(err, encapsulator.ErrNullVal)

	i, err = d.FromCty(cty.UnknownVal(e.CtyType()))
	assert.Nil(i)
	assert.ErrorIs(err, encapsulator.ErrUnknownVal)

	i, err = d.FromCty(cty.StringVal(""))
	assert.Nil(i)
	assert.ErrorIs(err, encapsulator.ErrWrongType)

	capsule := cty.Capsule("other type", reflect.TypeFor[string]())
	str := "hi"

	i, err = d.FromCty(cty.CapsuleVal(capsule, &str))
	assert.Nil(i)
	assert.ErrorIs(err, encapsulator.ErrWrongType)
}
