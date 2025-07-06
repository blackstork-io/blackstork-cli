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
	Items   *hclsyntax.Attribute

	Children []ContentTreeBlock

	DependsOn *hclsyntax.Attribute
}

func (d *Dynamic) isContentTreeBlock() {}

var _ ContentTreeBlock = (*Dynamic)(nil)
