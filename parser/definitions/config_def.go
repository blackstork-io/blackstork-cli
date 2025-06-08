package definitions

import (
	"context"
	"sync"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/parser/evaluation"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/encapsulator"
	"github.com/blackstork-io/fabric/plugin/dataspec"
)

// Configuration block definition.
type ConfigDef struct {
	*hclsyntax.Block
	once  sync.Once
	value *dataspec.Block
}

// Implements evaluation.Configuration.
func (c *ConfigDef) Exists() bool {
	return c != nil
}

var _ evaluation.Configuration = (*ConfigDef)(nil)

// ParseConfig implements Configuration.
func (c *ConfigDef) ParseConfig(ctx context.Context, spec *dataspec.RootSpec) (val *dataspec.Block, diags diagnostics.Diag) {
	c.once.Do(func() {
		var diag diagnostics.Diag
		c.value, diag = dataspec.DecodeAndEvalBlock(ctx, c.Block, spec, nil)
		if diags.Extend(diag) {
			// don't let partially-decoded values live
			c.value = nil
		}
	})
	val = c.value
	if val == nil && diags == nil {
		diags.Append(diagnostics.RepeatedError)
	}
	return
}

func (c *ConfigDef) Key() *Key {
	var name string
	switch len(c.Labels) {
	case 0:
		// anonymous config block
		return nil
	case 3:
		// named config block
		name = c.Labels[2]
		fallthrough
	case 2:
		// default config block
		return &Key{
			Kind: c.Labels[0],
			Runner: c.Labels[1],
			Name:  name,
		}
	default:
		panic("Invalid config block definition")
	}
}

func (c *ConfigDef) ApplicableTo(execBlock *ExecBlockDef) bool {
	switch len(c.Labels) {
	case 0:
		// anonymous config block
		return true
	case 2, 3:
		// named config block
		return execBlock.Kind() == c.Labels[0] && execBlock.Name() == c.Labels[1]
	default:
		panic("Config block definitions is invalid")
	}
}

func (c *ConfigDef) Kind() string {
	return BlockKindConfig
}

var _ BlockDef = (*ConfigDef)(nil)

func (c *ConfigDef) GetHCLBlock() *hclsyntax.Block {
	return c.Block
}

var ctyConfigType = encapsulator.NewEncoder[ConfigDef]("config", nil)

func (c *ConfigDef) CtyType() cty.Type {
	return ctyConfigType.CtyType()
}

func DefineConfigDef(block *hclsyntax.Block) (config *ConfigDef, diags diagnostics.Diag) {
	diags.Append(validateExecBlockKindLabel(block, 0))
	diags.Append(validateBlockRunnerName(block, 1))
	diags.Append(validateBlockName(block, 2, false))
	diags.Append(validateLabelsLength(block, 3, "plugin_kind plugin_name <block_name>"))

	if diags.HasErrors() {
		return
	}
	config = &ConfigDef{
		Block: block,
	}
	return
}

type ConfigResolver func(execBlockKind, blockRunnerName string) (config *ConfigDef)
