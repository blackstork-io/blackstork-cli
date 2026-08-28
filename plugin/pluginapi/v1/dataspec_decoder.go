// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package pluginapiv1

import (
	"fmt"

	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func decodeAttrSpec(src *AttrSpec) (*dataspec.AttrSpec, error) {
	t, err := decodeCtyType(src.GetType())
	if err != nil {
		return nil, err
	}
	def, err := decodeCtyValue(src.GetDefaultVal())
	if err != nil {
		return nil, err
	}
	ex, err := decodeCtyValue(src.GetExampleVal())
	if err != nil {
		return nil, err
	}
	oneof, err := utils.FnMapErr(src.GetOneOf(), decodeCtyValue)
	if err != nil {
		return nil, err
	}
	min, err := decodeCtyValue(src.GetMinInclusive())
	if err != nil {
		return nil, err
	}
	max, err := decodeCtyValue(src.GetMaxInclusive())
	if err != nil {
		return nil, err
	}
	return &dataspec.AttrSpec{
		Name:         src.GetName(),
		Type:         t,
		DefaultVal:   def,
		ExampleVal:   ex,
		Doc:          src.GetDoc(),
		Constraints:  constraint.Constraints(src.GetConstraints()),
		OneOf:        oneof,
		MinInclusive: min,
		MaxInclusive: max,
		Deprecated:   src.GetDeprecated(),
		Secret:       src.GetSecret(),
	}, nil
}

func decodeBlockSpec(src *BlockSpec) (*dataspec.BlockSpec, error) {
	blockSpecs, err := utils.FnMapErr(src.GetBlockSpecs(), decodeBlockSpec)
	if err != nil {
		return nil, err
	}
	attrSpecs, err := utils.FnMapErr(src.GetAttrSpecs(), decodeAttrSpec)
	if err != nil {
		return nil, err
	}
	matchers := src.GetHeadersSpec()
	header := make(dataspec.HeadersSpec, 0, len(matchers))
	for _, m := range matchers {
		switch matcher := m.GetMatcher().(type) {
		case *BlockSpec_NameMatcher_Exact_:
			header = append(header, dataspec.ExactMatcher(matcher.Exact.Matches))
		default:
			return nil, fmt.Errorf("unexpected matcher type: %T", matcher)
		}
	}

	return &dataspec.BlockSpec{
		Header:     header,
		Required:   src.GetRequired(),
		Repeatable: src.GetRepeatable(),

		Doc: src.GetDoc(),

		Blocks: blockSpecs,
		Attrs:  attrSpecs,

		AllowUnspecifiedBlocks:     src.GetAllowUnspecifiedBlocks(),
		AllowUnspecifiedAttributes: src.GetAllowUnspecifiedAttributes(),
	}, nil
}

func decodeAttr(src *Attr) (*dataspec.Attr, error) {
	val, err := decodeCtyValue(src.GetValue())
	if err != nil {
		return nil, err
	}
	return &dataspec.Attr{
		Name:       src.GetName(),
		NameRange:  decodeRange(src.GetNameRange()),
		Value:      val,
		ValueRange: decodeRange(src.GetValueRange()),
		Secret:     src.GetSecret(),
	}, nil
}

func decodeBlock(src *Block) (*dataspec.Block, error) {
	blocks, err := utils.FnMapErr(src.GetBlocks(), decodeBlock)
	if err != nil {
		return nil, err
	}
	attrs, err := utils.MapMapErr(src.GetAttributes(), decodeAttr)
	if err != nil {
		return nil, err
	}
	return &dataspec.Block{
		Header:        src.GetHeader(),
		HeaderRanges:  utils.FnMap(src.GetHeaderRanges(), decodeRange),
		Blocks:        blocks,
		Attrs:         attrs,
		ContentsRange: decodeRange(src.GetContentsRange()),
	}, err
}
