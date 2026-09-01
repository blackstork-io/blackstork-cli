// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

// Package dataspec defines schemas and decoded values for plugin blocks.
package dataspec

import (
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
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

// RootSpec is the subset of BlockSpec that represents a root block.
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
			if a.Constraints.Is(constraint.Required) && a.DefaultVal.IsNull() {
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
