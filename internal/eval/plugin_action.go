package eval

import (
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
)

type PluginAction struct {
	Source          definitions.ExecBlock
	BlockRunnerName string
	BlockName       string
	meta            *definitions.MetaBlock
	Config          *dataspec.Block
	Args            *dataspec.Block
}
