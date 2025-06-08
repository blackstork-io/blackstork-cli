package definitions

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type Section struct {
	Source *SectionDef

	BlockName string

	Meta  *MetaBlock
	Title *ContentBlock

	Content []ContentTreeBlock

	Vars *Vars

	IsIncluded *hclsyntax.Attribute

	DependsOnCombined    []*hclsyntax.Attribute
	RequiredVarsCombined []*hclsyntax.Attribute
}

func (s *Section) isContentTreeBlock() {}
func (s *Section) isRefTargetBlock()   {}

func (s *Section) GetSourceKind() string {
	return s.Source.Kind()
}

var _ ContentTreeBlock = (*Section)(nil)
var _ RefTargetBlock = (*Section)(nil)
