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
	"github.com/zclconf/go-cty/cty"
)

var InputTypes = []string{
	"string", "number", "bool", "datetime", "secret",
}

type InputBlock struct {
	Name string // set from a label
	Type string `hcl:"type"`

	Label        *string   `hcl:"label,optional"`
	Description  *string   `hcl:"description,optional"`
	DefaultValue cty.Value `hcl:"default_value,optional"`

	Block *hclsyntax.Block
}
