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

type PublishBlock struct {
	Source *ExecBlockDef

	RunnerName string
	BlockName  string

	Meta   *MetaBlock
	Config evaluation.Configuration

	Format *FormatBlock
}

func (b *PublishBlock) isExecBlock()       {}
func (b *PublishBlock) isDetachableBlock() {}
func (b *PublishBlock) isRefTargetBlock()  {}

func (b *PublishBlock) GetSource() *ExecBlockDef {
	return b.Source
}

func (b *PublishBlock) GetMeta() *MetaBlock {
	return b.Meta
}

func (b *PublishBlock) GetSourceKind() string {
	return b.Source.Kind()
}

func (b *PublishBlock) GetRunner() string {
	return b.RunnerName
}

func (b *PublishBlock) GetName() string {
	return b.BlockName
}

var (
	_ ExecBlock       = (*PublishBlock)(nil)
	_ DetachableBlock = (*PublishBlock)(nil)
	_ RefTargetBlock  = (*PublishBlock)(nil)
)
