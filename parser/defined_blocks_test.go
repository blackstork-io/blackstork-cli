// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package parser_test

import (
	"testing"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"

	definitions_mocks "github.com/blackstork-io/blackstork-cli/mocks/specs/definitions"
	"github.com/blackstork-io/blackstork-cli/parser"
)

func TestAddIfMissing(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	m := map[string]*definitions_mocks.BlockDef{}

	m1 := definitions_mocks.NewBlockDef(t)
	m1.EXPECT().Kind().Return("content").Once()
	m1.EXPECT().GetHCLBlock().Return(&hclsyntax.Block{})

	diag := parser.AddIfMissing(m, "key_1", m1)
	assert.Empty(diag)
	assert.Same(m1, m["key_1"])

	m2 := definitions_mocks.NewBlockDef(t)

	diag = parser.AddIfMissing(m, "key_2", m2)
	assert.Empty(diag)
	assert.Same(m1, m["key_1"])
	assert.Same(m2, m["key_2"])

	m3 := definitions_mocks.NewBlockDef(t)
	m3.EXPECT().GetHCLBlock().Return(&hclsyntax.Block{}).Once()

	diag = parser.AddIfMissing(m, "key_1", m3)
	assert.NotEmpty(diag)
	assert.Same(m1, m["key_1"])
	assert.Same(m2, m["key_2"])
}
