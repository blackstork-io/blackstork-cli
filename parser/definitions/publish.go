package definitions

import (
	"github.com/blackstork-io/fabric/parser/evaluation"
)

type PublishBlock struct {
	Source *ExecBlockDef

	BlockRunnerName string
	BlockName       string

	Meta   *MetaBlock
	Config evaluation.Configuration
}

func (b *PublishBlock) isExecBlock() {}
func (b *PublishBlock) isRefTargetBlock()   {}

func (b *PublishBlock) GetSource() *ExecBlockDef {
	return b.Source
}

func (b *PublishBlock) GetMeta() *MetaBlock {
	return b.Meta
}

func (b *PublishBlock) GetSourceKind() string {
	return b.Source.Kind()
}

func (b *PublishBlock) GetRunner() string {
	return b.BlockRunnerName
}

func (b *PublishBlock) GetName() string {
	return b.BlockName
}

var _ ExecBlock = (*PublishBlock)(nil)
var _ RefTargetBlock = (*PublishBlock)(nil)
