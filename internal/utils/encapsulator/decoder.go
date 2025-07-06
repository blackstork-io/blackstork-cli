package encapsulator

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/zclconf/go-cty/cty"
)

var (
	ErrNullVal    = errors.New("null value")
	ErrUnknownVal = errors.New("unknown value")
	ErrWrongType  = errors.New("wrong type")
)

// Non-generic decoder interface
type DecoderI interface {
	Decodable(ty cty.Type) bool
	GoType() reflect.Type
	TypeEqual(ty cty.Type) bool
	ValDecodable(v cty.Value) bool
	ValTypeEqual(v cty.Value) bool
	fromCty(v cty.Value) (any, error)
	mustFromCty(v cty.Value) any
}

type decoderCore struct {
	goType reflect.Type
}

// Decodable checks if the given cty type can be decoded into T.
// This is a loose check, for example it allows decoding value that implements interface T.
func (d *decoderCore) Decodable(ty cty.Type) bool {
	return ty.IsCapsuleType() && reflect.PointerTo(ty.EncapsulatedType()).AssignableTo(d.goType)
}

// ValDecodable checks if the given cty value can be decoded into T.
// This is a loose check, for example it allows decoding value that implements interface T.
func (d *decoderCore) ValDecodable(v cty.Value) bool {
	v, _ = v.Unmark()
	return isValid(v) && d.Decodable(v.Type())
}

// TypeEqual checks if the given cty type can be decoded into T.
// This is a stricter check, requiring that encapsulated type is exactly T.
func (d *decoderCore) TypeEqual(ty cty.Type) bool {
	return ty.IsCapsuleType() && reflect.PointerTo(ty.EncapsulatedType()) == d.goType
}

// ValTypeEqual checks if the given cty value can be decoded into T.
// This is a stricter check, requiring that encapsulated type is exactly T.
func (d *decoderCore) ValTypeEqual(v cty.Value) bool {
	v, _ = v.Unmark()
	return isValid(v) && d.TypeEqual(v.Type())
}

// GoType returns the go type of the decoded values.
func (d *decoderCore) GoType() reflect.Type {
	return d.goType
}

func (d *decoderCore) fromCty(v cty.Value) (any, error) {
	v, _ = v.Unmark()
	if v.IsNull() {
		return nil, ErrNullVal
	}
	if !v.IsKnown() {
		return nil, ErrUnknownVal
	}
	ty := v.Type()
	if !ty.IsCapsuleType() {
		return nil, fmt.Errorf(
			"%w: expected encapsulated %s, got %s",
			ErrWrongType, d.goType, ty.FriendlyName(),
		)
	}
	ety := reflect.PointerTo(ty.EncapsulatedType())
	if !ety.AssignableTo(d.goType) {
		return nil, fmt.Errorf(
			"%w: expected assignable to %s, got %s",
			ErrWrongType, d.goType, ety,
		)
	}
	return v.EncapsulatedValue(), nil
}

func (d *decoderCore) mustFromCty(v cty.Value) any {
	v, _ = v.Unmark()
	return v.EncapsulatedValue()
}

// Decoder extracts the encapsulated value from cty.Value.
type Decoder[T any] struct {
	decoderCore
}

// FromCty decodes the encapsulated T from cty.Value.
// Returns an error if the value can't be decoded into T.
func (d *Decoder[T]) FromCty(v cty.Value) (result T, err error) {
	var res any
	res, err = d.fromCty(v)
	if err == nil {
		result = res.(T)
	}
	return
}

// MustFromCty decodes the encapsulated T from cty.Value.
// Panics if the value can't be decoded into T.
func (d *Decoder[T]) MustFromCty(v cty.Value) T {
	return d.mustFromCty(v).(T)
}

func NewDecoder[T any]() *Decoder[T] {
	return &Decoder[T]{
		decoderCore: decoderCore{
			goType: reflect.TypeFor[T](),
		},
	}
}
