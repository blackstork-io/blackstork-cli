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
	"context"
	"strings"
	"sync"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/parser/evaluation"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils/encapsulator"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

// ConfigDef is configuration block definition.
type ConfigDef struct {
	*hclsyntax.Block
	once  sync.Once
	value *dataspec.Block
}

// Exists implements evaluation.Configuration.
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
	return val, diags
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
			Kind:   c.Labels[0],
			Runner: c.Labels[1],
			Name:   name,
		}
	default:
		panic("Invalid config block definition")
	}
}

func (c *ConfigDef) ForKind() string {
	switch len(c.Labels) {
	case 0:
		// anonymous config block
		return ""
	case 2, 3:
		targetBlockKind := c.Labels[0]
		targetBlockRunner := c.Labels[1]
		return strings.Join([]string{targetBlockKind, targetBlockRunner}, ".")
	default:
		panic("Config block definitions is invalid")
	}
}

func (c *ConfigDef) ApplicableTo(execBlock *ExecBlockDef) bool {
	switch len(c.Labels) {
	case 0:
		// anonymous config block
		return true
	case 2, 3:
		// named config block
		// return execBlock.Kind() == c.Labels[0] && execBlock.Name() == c.Labels[1]
		targetBlockKind := c.Labels[0]
		targetBlockRunner := c.Labels[1]
		return execBlock.Kind() == targetBlockKind && execBlock.RunnerName() == targetBlockRunner
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
	diags.Append(validateRunnerName(block, 1))
	diags.Append(validateBlockName(block, 2, false))
	diags.Append(validateLabelsLength(block, 3, "plugin_kind plugin_name <block_name>"))

	if diags.HasErrors() {
		return config, diags
	}
	config = &ConfigDef{
		Block: block,
	}
	return config, diags
}

type ConfigResolver func(execBlockKind, blockRunnerName string) (config *ConfigDef)
