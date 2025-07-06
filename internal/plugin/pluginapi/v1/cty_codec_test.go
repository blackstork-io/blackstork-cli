package pluginapiv1

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

func roundTrip(t *testing.T, val cty.Value) (decVal cty.Value, decTy cty.Type) {
	t.Helper()
	require := require.New(t)

	encVal, diag := encodeCtyValue(val)
	require.False(diag.HasErrors(), "Failed to encode value", diag)
	decVal, err := decodeCtyValue(encVal)
	require.NoError(err, "Failed to decode value")
	require.True(decVal.RawEquals(val), "Roundtrip failed: got", decVal.GoString())
	decTy = roundTripType(t, val.Type())
	return
}

func roundTripType(t *testing.T, ty cty.Type) (decTy cty.Type) {
	t.Helper()
	require := require.New(t)

	encTy, diag := encodeCtyType(ty)
	require.False(diag.HasErrors(), "Failed to encode type", diag)
	decTy, err := decodeCtyType(encTy)
	require.NoError(err, "Failed to decode type")
	require.True(decTy.Equals(ty), "Roundtrip failed: got", decTy.GoString())
	return
}

func TestCtyCodecRoundtrip(t *testing.T) {
	vals := []cty.Value{
		// all public cty.Value vars in cty
		cty.DynamicVal,
		cty.EmptyObjectVal,
		cty.EmptyTupleVal,
		cty.False,
		cty.NegativeInfinity,
		cty.PositiveInfinity,
		cty.True,
		cty.Zero,

		cty.StringVal("hello"),
		cty.StringVal(""),
		cty.NumberIntVal(42),
		cty.NumberIntVal(0),
		cty.NumberFloatVal(3.14),
		cty.NumberFloatVal(0),

		cty.ObjectVal(map[string]cty.Value{
			"foo": cty.StringVal("bar"),
		}),
		cty.ObjectVal(map[string]cty.Value{}),
		cty.ObjectVal(map[string]cty.Value{
			"foo": cty.StringVal("bar"),
			"baz": cty.NumberIntVal(42),
			"bar": cty.True,
		}),
		cty.ObjectVal(map[string]cty.Value{
			"foo": cty.NullVal(cty.String),
			"baz": cty.UnknownVal(cty.Number),
			"bar": cty.DynamicVal,
		}),
		cty.ObjectVal(map[string]cty.Value{
			"foo": cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
				"bar": cty.DynamicVal,
			}),
			"bar": cty.NullVal(cty.Object(map[string]cty.Type{
				"foo": cty.String,
				"bar": cty.DynamicPseudoType,
			})),
		}),

		cty.MapVal(map[string]cty.Value{
			"foo": cty.StringVal("bar"),
			"baz": cty.StringVal("foo"),
			"bar": cty.StringVal("baz"),
		}),
		cty.MapVal(map[string]cty.Value{
			"foo": cty.StringVal("bar"),
			"baz": cty.NullVal(cty.String),
			"bar": cty.UnknownVal(cty.String),
		}),
		cty.MapValEmpty(cty.Number),
		cty.MapValEmpty(cty.Map(cty.Tuple([]cty.Type{cty.Bool, cty.String}))),
		cty.MapValEmpty(cty.DynamicPseudoType),

		cty.ListVal([]cty.Value{
			cty.StringVal("foo"),
			cty.StringVal("foo"),
			cty.StringVal("bar"),
			cty.StringVal("bar"),
		}),
		cty.ListValEmpty(cty.String),
		cty.ListVal([]cty.Value{
			cty.NumberIntVal(42),
			cty.NumberIntVal(43),
		}),
		cty.ListValEmpty(cty.Number),
		cty.ListVal([]cty.Value{
			cty.NumberIntVal(42),
			cty.NumberIntVal(42),
			cty.NullVal(cty.Number),
			cty.UnknownVal(cty.Number),
		}),

		cty.SetVal([]cty.Value{
			cty.StringVal("foo"),
			cty.StringVal("foo"),
			cty.StringVal("bar"),
			cty.StringVal("bar"),
		}),
		cty.SetValEmpty(cty.String),
		cty.SetVal([]cty.Value{
			cty.NumberIntVal(42),
			cty.NumberIntVal(43),
		}),
		cty.SetValEmpty(cty.Number),
		cty.SetVal([]cty.Value{
			cty.NumberIntVal(42),
			cty.NumberIntVal(42),
			cty.NullVal(cty.Number),
			cty.UnknownVal(cty.Number),
		}),

		cty.TupleVal([]cty.Value{
			cty.StringVal("foo"),
			cty.NumberIntVal(42),
			cty.True,
			cty.DynamicVal,
		}),
		cty.TupleVal([]cty.Value{
			cty.True,
			cty.NullVal(cty.String),
			cty.UnknownVal(cty.Number),
		}),
		cty.TupleVal([]cty.Value{}),
	}
	for i, val := range vals {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Logf("Testing value roundtrip: %s", val.GoString())
			roundTrip(t, val)
			t.Logf("Testing UnknownVal roundtrip")
			roundTrip(t, cty.UnknownVal(val.Type()))
			t.Logf("Testing NullVal roundtrip")
			roundTrip(t, cty.NullVal(val.Type()))
		})
	}
}

func TestCtyCodecNil(t *testing.T) {
	var err error
	require := require.New(t)

	t.Logf("Testing Nil roundtrip")
	encVal, diag := encodeCtyValue(cty.NilVal)
	require.False(diag.HasErrors(), "Failed to encode value", diag)
	decVal, err := decodeCtyValue(encVal)
	require.NoError(err, "Failed to decode value")

	if decVal != cty.NilVal {
		t.Fatalf("Value roundtrip failed: got %s", decVal.GoString())
	}

	t.Logf("Testing NilType roundtrip")
	encTy, diag := encodeCtyType(cty.NilType)
	if diag.HasErrors() {
		t.Fatalf("Failed to encode type: %s", diag)
	}
	decTy, err := decodeCtyType(encTy)
	if err != nil {
		t.Fatalf("Failed to decode type: %s", err)
	}
	if decTy != cty.NilType {
		t.Fatalf("Type roundtrip failed: got %s", decTy.GoString())
	}
	return
}

func TestCtyCodecObjectOptional(t *testing.T) {
	require := require.New(t)

	ty := cty.ObjectWithOptionalAttrs(
		map[string]cty.Type{
			"required": cty.String,
			"optional": cty.String,
		},
		[]string{"optional"},
	)
	require.True(ty.AttributeOptional("optional"))

	decTy := roundTripType(t, ty)
	require.True(decTy.AttributeOptional("optional"))
}

func TestCtyCodecCapsule(t *testing.T) {
	require := require.New(t)

	var err error
	t.Logf("Testing Capsule roundtrip")

	data := plugindata.Data(plugindata.Map{
		"foo": plugindata.String("bar"),
		"bar": plugindata.Number(42),
		"baz": plugindata.Bool(true),
		"qux": plugindata.List{
			plugindata.String("foo"),
			plugindata.String("bar"),
		},
	})

	ctyVal := plugindata.Encapsulated.ToCty(&data)
	encVal, diag := encodeCtyValue(ctyVal)
	require.False(diag.HasErrors(), "Failed to encode value", diag)

	decVal, err := decodeCtyValue(encVal)
	require.NoError(err, "Failed to decode value")

	decData := plugindata.Encapsulated.MustFromCty(decVal)
	require.Equal(&data, decData)
	roundTripType(t, plugindata.Encapsulated.CtyType())
}
