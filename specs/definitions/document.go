// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package definitions

type Document struct {
	Source       *DocumentDef
	Meta         *MetaBlock
	Inputs       []*InputBlock
	Vars         *Vars
	RequiredVars []string

	DataBlocks        []*DataBlock
	Title             ContentTreeBlock
	ContentTreeBlocks []ContentTreeBlock
	FormatBlocks      []*FormatBlock
	PublishBlocks     []*PublishBlock
}

func (b *Document) isRefTargetBlock() {}
func (b *Document) GetSourceKind() string {
	return b.Source.Kind()
}

var _ RefTargetBlock = (*Document)(nil)
