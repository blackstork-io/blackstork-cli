// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package definitions

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils/encapsulator"
)

type ExecBlockDef struct {
	Block *hclsyntax.Block

	// Once        sync.Once
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

var (
	_ BlockDef     = (*ExecBlockDef)(nil)
	_ RefSourceDef = (*ExecBlockDef)(nil)
)

func (p *ExecBlockDef) GetHCLBlock() *hclsyntax.Block {
	return p.Block
}

var ctyExecBlockDefType = encapsulator.NewEncoder[ExecBlockDef]("exec_block_def", nil)

func (p *ExecBlockDef) CtyType() cty.Type {
	return ctyExecBlockDefType.CtyType()
}

func DefineExecBlockDef(block *hclsyntax.Block, atTopLevel bool) (plugin *ExecBlockDef, diags diagnostics.Diag) {
	nameRequired := atTopLevel

	diags.Append(validateExecBlockKind(block, block.Type, block.TypeRange))
	diags.Append(validateRunnerName(block, 0))
	diags.Append(validateBlockName(block, 1, nameRequired))

	var usage string
	if nameRequired {
		usage = "block_runner_name block_name"
	} else {
		usage = "block_runner_name <block_name>"
	}
	diags.Append(validateLabelsLength(block, 2, usage))

	if diags.HasErrors() {
		return plugin, diags
	}

	plugin = &ExecBlockDef{
		Block: block,
	}

	return plugin, diags
}
