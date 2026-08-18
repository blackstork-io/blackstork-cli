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
	"reflect"

	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/parser/evaluation"
)

type FormatBlock struct {
	Source     *ExecBlockDef
	RunnerName string
	BlockName  string

	Meta   *MetaBlock
	Config evaluation.Configuration
}

func (b *FormatBlock) isExecBlock()       {}
func (b *FormatBlock) isDetachableBlock() {}
func (b *FormatBlock) isRefTargetBlock()  {}

func (b *FormatBlock) GetSource() *ExecBlockDef {
	return b.Source
}

func (b *FormatBlock) GetSourceKind() string {
	return b.Source.Kind()
}

func (b *FormatBlock) GetMeta() *MetaBlock {
	return b.Meta
}

func (b *FormatBlock) GetRunner() string {
	return b.RunnerName
}

func (b *FormatBlock) GetName() string {
	return b.BlockName
}

var ctyFormatBlockType = cty.Capsule("format_block", reflect.TypeFor[FormatBlock]())

func (b *FormatBlock) CtyType() cty.Type {
	return ctyFormatBlockType
}

func (b *FormatBlock) CtyValue() cty.Value {
	return cty.CapsuleVal(b.CtyType(), b)
}

var (
	_ ExecBlock       = (*FormatBlock)(nil)
	_ DetachableBlock = (*FormatBlock)(nil)
	_ RefTargetBlock  = (*FormatBlock)(nil)
)
