package definitions

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/fabric/parser/evaluation"
)

type ContentBlock struct {
	Source *ExecBlockDef

	BlockRunnerName string
	BlockName       string

	Meta   *MetaBlock
	Config evaluation.Configuration

	Vars *Vars

	IsIncluded *hclsyntax.Attribute

	DependsOnCombined    []*hclsyntax.Attribute
	RequiredVarsCombined []*hclsyntax.Attribute
}

func (b *ContentBlock) isContentTreeBlock() {}
func (b *ContentBlock) isExecBlock()        {}
func (b *ContentBlock) isRefTargetBlock()   {}

func (b *ContentBlock) GetSource() *ExecBlockDef {
	return b.Source
}

func (b *ContentBlock) GetSourceKind() string {
	return b.Source.Kind()
}

//
// func (b ContentBlock) GetBlockName() string {
// 	return b.BlockName
// }
//
// func (b ContentBlock) GetRunnerName() string {
// 	return b.BlockRunnerName
// }
//
// func (b ContentBlock) GetConfig() evaluation.Configuration {
// 	return b.Config
// }

var _ ContentTreeBlock = (*ContentBlock)(nil)
var _ ExecBlock = (*ContentBlock)(nil)
var _ RefTargetBlock = (*ContentBlock)(nil)
