package definitions

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/encapsulator"
)

type ExecBlockDef struct {
	Block *hclsyntax.Block

	//Once        sync.Once
}

func (p *ExecBlockDef) DefRange() hcl.Range {
	return p.Block.DefRange()
}

func (p *ExecBlockDef) Kind() string {
	return p.Block.Type
}

func (p *ExecBlockDef) RunnerName() string {
	// 	if p.Parsed {
	// 		// resolved block name (in case of ref)
	// 		return p.ParseResult.BlockName
	// 	}
	return p.Block.Labels[0]
}

// Whether or not the original block is a ref.
func (p *ExecBlockDef) IsRef() bool {
	return p.Block.Labels[0] == BlockTypeRef
}

func (p *ExecBlockDef) Name() string {
	if len(p.Block.Labels) < 2 {
		return ""
	}
	return p.Block.Labels[1]
}

func (p *ExecBlockDef) Key() Key {
	return Key{
		Kind:   p.Kind(),
		Runner: p.RunnerName(),
		Name:   p.Name(),
	}
}

var _ BlockDef = (*ExecBlockDef)(nil)
var _ RefSourceDef = (*ExecBlockDef)(nil)

func (p *ExecBlockDef) GetHCLBlock() *hclsyntax.Block {
	return p.Block
}

var ctyExecBlockType = encapsulator.NewEncoder[ExecBlockDef]("exec_block", nil)

func (p *ExecBlockDef) CtyType() cty.Type {
	return ctyExecBlockType.CtyType()
}

func DefineExecBlockDef(block *hclsyntax.Block, atTopLevel bool) (plugin *ExecBlockDef, diags diagnostics.Diag) {
	nameRequired := atTopLevel

	diags.Append(validateExecBlockKind(block, block.Type, block.TypeRange))
	diags.Append(validateBlockRunnerName(block, 0))
	diags.Append(validateBlockName(block, 1, nameRequired))

	var usage string
	if nameRequired {
		usage = "block_runner_name block_name"
	} else {
		usage = "block_runner_name <block_name>"
	}
	diags.Append(validateLabelsLength(block, 2, usage))

	if diags.HasErrors() {
		return
	}

	plugin = &ExecBlockDef{
		Block: block,
	}

	return
}
