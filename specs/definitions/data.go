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
	"github.com/blackstork-io/blackstork-cli/parser/evaluation"
)

type DataBlock struct {
	Source     *ExecBlockDef
	RunnerName string
	BlockName  string

	Meta   *MetaBlock
	Config evaluation.Configuration
}

func (b *DataBlock) isContentTreeBlock() {}
func (b *DataBlock) isExecBlock()        {}
func (b *DataBlock) isDetachableBlock()  {}
func (b *DataBlock) isRefTargetBlock()   {}

func (b *DataBlock) GetSource() *ExecBlockDef {
	return b.Source
}

func (b *DataBlock) GetSourceKind() string {
	return b.Source.Kind()
}

func (b *DataBlock) GetMeta() *MetaBlock {
	return b.Meta
}

func (b *DataBlock) GetRunner() string {
	return b.RunnerName
}

func (b *DataBlock) GetName() string {
	return b.BlockName
}

var (
	_ ContentTreeBlock = (*DataBlock)(nil)
	_ ExecBlock        = (*DataBlock)(nil)
	_ DetachableBlock  = (*DataBlock)(nil)
	_ RefTargetBlock   = (*DataBlock)(nil)
)
