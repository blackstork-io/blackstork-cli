package definitions

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type RefSourceDef interface {
	IsRef() bool
	Kind() string
	GetHCLBlock() *hclsyntax.Block
	DefRange() hcl.Range
}
