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

type DynamicDef struct {
	Block *hclsyntax.Block
}

func (d *DynamicDef) Name() string {
	if len(d.Block.Labels) < 2 {
		return ""
	}
	return d.Block.Labels[1]
}

type Dynamic struct {
	Source *DynamicDef

	BlockName string

	// Items is a list of items to be iterated over dynamically. Always present.
	Items *hclsyntax.Attribute

	Children []ContentTreeBlock

	DependsOn *hclsyntax.Attribute
}

func (d *Dynamic) isContentTreeBlock() {}

var _ ContentTreeBlock = (*Dynamic)(nil)
