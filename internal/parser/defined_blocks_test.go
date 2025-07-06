package parser_test

import (
	"testing"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"

	definitions_mocks "github.com/blackstork-io/fabric/mocks/parser/definitions"
	"github.com/blackstork-io/fabric/parser"
)

func TestAddIfMissing(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	m := map[string]*definitions_mocks.FabricBlock{}

	m1 := definitions_mocks.NewFabricBlock(t)
	m1.EXPECT().GetHCLBlock().Return(&hclsyntax.Block{})

	diag := parser.AddIfMissing(m, "key_1", m1)
	assert.Empty(diag)
	assert.Same(m1, m["key_1"])

	m2 := definitions_mocks.NewFabricBlock(t)

	diag = parser.AddIfMissing(m, "key_2", m2)
	assert.Empty(diag)
	assert.Same(m1, m["key_1"])
	assert.Same(m2, m["key_2"])

	m3 := definitions_mocks.NewFabricBlock(t)
	m3.EXPECT().GetHCLBlock().Return(&hclsyntax.Block{}).Once()

	diag = parser.AddIfMissing(m, "key_1", m3)
	assert.NotEmpty(diag)
	assert.Same(m1, m["key_1"])
	assert.Same(m2, m["key_2"])
}
