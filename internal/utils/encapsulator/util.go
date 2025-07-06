// Type-safe conversions to and from cty.CapsuleValue
package encapsulator

import (
	"github.com/zclconf/go-cty/cty"
)

// isValid tests that cty.Value is safe to work with.
func isValid(v cty.Value) bool {
	return !v.IsNull() && v.IsKnown()
}

// Compatible checks that values produced by the given Encoder are decodable by the given Decoder.
// This is a loose check, for example it allows decoding value that implements interface DT.
func Compatible(encoder EncoderI, decoder DecoderI) bool {
	return decoder.Decodable(encoder.CtyType())
}
