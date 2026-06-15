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

	"github.com/blackstork-io/blackstork-cli/parser/evaluation"
)

type ContentBlock struct {
	Source *ExecBlockDef

	RunnerName string
	BlockName  string

	Meta   *MetaBlock
	Config evaluation.Configuration

	Vars *Vars

	IsIncluded *hclsyntax.Attribute

	DependsOnCombined    []*hclsyntax.Attribute
	RequiredVarsCombined []*hclsyntax.Attribute
}

func (b *ContentBlock) isContentTreeBlock() {}
func (b *ContentBlock) isExecBlock()        {}
func (b *ContentBlock) isDetachableBlock()  {}
func (b *ContentBlock) isRefTargetBlock()   {}

func (b *ContentBlock) GetSource() *ExecBlockDef {
	return b.Source
}

func (b *ContentBlock) GetSourceKind() string {
	return b.Source.Kind()
}

func (b *ContentBlock) GetMeta() *MetaBlock {
	return b.Meta
}

func (b *ContentBlock) GetRunner() string {
	return b.RunnerName
}

func (b *ContentBlock) GetName() string {
	return b.BlockName
}

var (
	_ ContentTreeBlock = (*ContentBlock)(nil)
	_ ExecBlock        = (*ContentBlock)(nil)
	_ DetachableBlock  = (*ContentBlock)(nil)
	_ RefTargetBlock   = (*ContentBlock)(nil)
)
