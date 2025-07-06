package encapsulator

import (
	"reflect"
)

// Codec is both an Encoder and a Decoder for *T.
type Codec[T any] struct {
	Decoder[*T]
	Encoder[T]
}

// NewCodec creates a Codec (encoder + decoder).
// Will use cty.Capsule if capsuleOps is nil, and cty.CapsuleWithOps otherwise
func NewCodec[T any](friendlyName string, capsuleOps *CapsuleOps[T]) *Codec[T] {
	goType := reflect.TypeFor[T]()
	codec := &Codec[T]{
		Encoder: Encoder[T]{},
		Decoder: Decoder[*T]{
			decoderCore: decoderCore{
				goType: reflect.PointerTo(goType),
			},
		},
	}
	codec.initEncoderCore(friendlyName, goType, capsuleOps)
	return codec
}
