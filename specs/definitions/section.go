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
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type Section struct {
	Source *SectionDef

	BlockName string

	Meta  *MetaBlock
	Title *ContentBlock

	Content []ContentTreeBlock

	Vars *Vars

	IsIncluded *hclsyntax.Attribute

	DependsOnCombined    []*hclsyntax.Attribute
	RequiredVarsCombined []*hclsyntax.Attribute
}

func (s *Section) isContentTreeBlock() {}
func (s *Section) isDetachableBlock()  {}
func (s *Section) isRefTargetBlock()   {}
func (s *Section) GetRunner() string {
	return ""
}

func (s *Section) GetSourceKind() string {
	return s.Source.Kind()
}

func (s *Section) GetMeta() *MetaBlock {
	return s.Meta
}

var (
	_ ContentTreeBlock = (*Section)(nil)
	_ DetachableBlock  = (*Section)(nil)
	_ RefTargetBlock   = (*Section)(nil)
)
