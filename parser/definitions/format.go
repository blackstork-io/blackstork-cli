package definitions

import (
	"github.com/blackstork-io/fabric/parser/evaluation"
)

type FormatBlock struct {
	Source          *ExecBlockDef
	BlockRunnerName string
	BlockName       string

	Meta   *MetaBlock
	Config evaluation.Configuration
}

func (b *FormatBlock) isExecBlock()      {}
func (b *FormatBlock) isRefTargetBlock() {}

func (b *FormatBlock) GetSource() *ExecBlockDef {
	return b.Source
}

func (b *FormatBlock) GetSourceKind() string {
	return b.Source.Kind()
}

var _ ExecBlock = (*FormatBlock)(nil)
var _ RefTargetBlock = (*FormatBlock)(nil)
