package constraint

import (
	"math"
	"math/big"
	"regexp"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

type Constraints uint32

func (c Constraints) Is(test Constraints) bool {
	return (c & test) == test
}

const (
	// Attribute can't be left out (but can be null, empty, etc.)
	Required Constraints = (1 << iota)
	// Attribute can't be null
	NonNull
	// Attribute can't be an empty collection or an empty string
	NonEmpty
	// If an attribute is a string - preprocess it with strings.TrimSpace
	TrimSpace
	// Attribute must be a whole integer number
	Integer
)

const (
	// Attribute will be trimmed and later checked to be non-empty
	// If attribute is not a string - trim operation does nothing
	TrimmedNonEmpty Constraints = TrimSpace | NonEmpty

	// Attribute is not required, but (if specified) must be non-null, non-empty, strings are trimmed
	Meaningful Constraints = NonNull | TrimmedNonEmpty

	// Attribute is required, non-null, non-empty, strings are trimmed
	RequiredMeaningful Constraints = Required | Meaningful

	// Attribute is required and non-null
	RequiredNonNull Constraints = Required | NonNull
)

type OneOf []cty.Value

var oneOfRe = regexp.MustCompile(`(?s)^\s*t\s*=\s*\[\s*(.*?)\s*\]\s*$`)

func (o OneOf) String() string {
	f := hclwrite.NewFile()
	f.Body().SetAttributeValue("t", cty.TupleVal(o))

	b := hclwrite.Format(f.Bytes())
	return string(oneOfRe.FindSubmatch(b)[1])
}

func (o OneOf) IsEmpty() bool {
	return len(o) == 0
}

func (o OneOf) Validate(val cty.Value) (valid bool) {
	if len(o) == 0 {
		return true
	}
	for _, el := range o {
		if el.Equals(val).True() {
			return true
		}
	}
	return false
}

var (
	// math.MaxInt as a big.Float
	MaxInt = new(big.Float).SetInt64(math.MaxInt)
	// math.MaxInt as a cty number
	MaxIntVal = cty.NumberVal(MaxInt)
	// math.MinInt as a big.Float
	MinInt = new(big.Float).SetInt64(math.MinInt)
	// math.MinInt as a cty number
	MinIntVal = cty.NumberVal(MinInt)
	// math.MaxInt64 as a big.Float
	MaxInt64 = new(big.Float).SetInt64(math.MaxInt64)
	// math.MaxInt64 as a cty number
	MaxInt64Val = cty.NumberVal(MaxInt64)
	// math.MinInt64 as a big.Float
	MinInt64 = new(big.Float).SetInt64(math.MinInt64)
	// math.MinInt64 as a cty number
	MinInt64Val = cty.NumberVal(MinInt64)
)
