// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package definitions

import (
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils/encapsulator"
)

type SectionDef struct {
	Block       *hclsyntax.Block
	Once        sync.Once
	Parsed      bool
	ParseResult *Section
}

func (s *SectionDef) IsRef() bool {
	return len(s.Block.Labels) > 0 && s.Block.Labels[0] == BlockTypeRef
}

// The index of a name label in the labels list
func (s *SectionDef) nameIdx() int {
	if s.IsRef() {
		return 1
	}
	return 0
}

func (s *SectionDef) Name() string {
	nameIdx := s.nameIdx()
	if len(s.Block.Labels) > nameIdx {
		return s.Block.Labels[nameIdx]
	}
	return ""
}

func (s *SectionDef) Kind() string {
	return BlockKindSection
}

func (p *SectionDef) DefRange() hcl.Range {
	return p.Block.DefRange()
}

var (
	_ BlockDef     = (*SectionDef)(nil)
	_ RefSourceDef = (*SectionDef)(nil)
)

func (s *SectionDef) GetHCLBlock() *hclsyntax.Block {
	return s.Block
}

var ctySectionType = encapsulator.NewEncoder[SectionDef]("section", nil)

func (*SectionDef) CtyType() cty.Type {
	return ctySectionType.CtyType()
}

func DefineSectionDef(block *hclsyntax.Block, atTopLevel bool) (section *SectionDef, diags diagnostics.Diag) {
	sect := SectionDef{
		Block: block,
	}

	nameRequired := atTopLevel

	labels := "<ref> "
	if nameRequired {
		labels += "block_name"
	} else {
		labels += "<block_name>"
	}

	diags.Append(validateBlockName(block, sect.nameIdx(), nameRequired))
	diags.Append(validateLabelsLength(block, 2, labels))
	if diags.HasErrors() {
		return section, diags
	}

	section = &sect
	return section, diags
}
