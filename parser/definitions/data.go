package definitions

import (
	"github.com/blackstork-io/fabric/parser/evaluation"
)

type DataBlock struct {
	Source          *ExecBlockDef
	BlockRunnerName string
	BlockName       string
	Meta            *MetaBlock
	Config          evaluation.Configuration
}

func (b *DataBlock) isContentTreeBlock() {}
func (b *DataBlock) isExecBlock()        {}
func (b *DataBlock) isRefTargetBlock()   {}

func (b *DataBlock) GetSource() *ExecBlockDef {
	return b.Source
}

func (b *DataBlock) GetSourceKind() string {
	return b.Source.Kind()
}

func (b *DataBlock) GetMeta() *MetaBlock {
	return b.Meta
}

func (b *DataBlock) GetRunner() string {
	return b.BlockRunnerName
}

func (b *DataBlock) GetName() string {
	return b.BlockName
}

var _ ContentTreeBlock = (*DataBlock)(nil)
var _ ExecBlock = (*DataBlock)(nil)
var _ RefTargetBlock = (*DataBlock)(nil)
