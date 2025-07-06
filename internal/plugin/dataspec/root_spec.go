package dataspec

import (
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/constraint"
)

func RootSpecFromBlock(b *BlockSpec) *RootSpec {
	if b == nil {
		return nil
	}
	return &RootSpec{
		Doc:                        b.Doc,
		Blocks:                     b.Blocks,
		Attrs:                      b.Attrs,
		AllowUnspecifiedBlocks:     b.AllowUnspecifiedBlocks,
		AllowUnspecifiedAttributes: b.AllowUnspecifiedAttributes,
		Required:                   b.Required,
		blockSpec:                  b,
	}
}

// A subset of BlockSpec that represents the root block.
type RootSpec struct {
	Doc string

	Blocks []*BlockSpec
	Attrs  []*AttrSpec

	Required                   bool
	AllowUnspecifiedBlocks     bool
	AllowUnspecifiedAttributes bool
	blockSpec                  *BlockSpec
}

func (r *RootSpec) IsRequired() bool {
	if r != nil {
		return r.BlockSpec().Required
	}
	return false
}

func (r *RootSpec) BlockSpec() *BlockSpec {
	if r == nil {
		return nil
	}
	if r.blockSpec == nil {
		r.makeBlockSpec()
	}
	return r.blockSpec
}

func (r *RootSpec) makeBlockSpec() {
	isRequired := r.Required
	if !isRequired {
		for _, b := range r.Blocks {
			if b.Required {
				isRequired = true
				break
			}
		}
	}
	if !isRequired {
		for _, a := range r.Attrs {
			if a.Constraints.Is(constraint.Required) {
				isRequired = true
				break
			}
		}
	}
	r.blockSpec = &BlockSpec{
		Required:                   isRequired,
		Repeatable:                 false,
		Doc:                        r.Doc,
		Blocks:                     r.Blocks,
		Attrs:                      r.Attrs,
		AllowUnspecifiedBlocks:     r.AllowUnspecifiedBlocks,
		AllowUnspecifiedAttributes: r.AllowUnspecifiedAttributes,
	}
}

func (r *RootSpec) ValidateSpec() (errs diagnostics.Diag) {
	return r.BlockSpec().ValidateSpec()
}
