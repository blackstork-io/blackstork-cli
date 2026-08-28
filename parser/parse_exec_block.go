// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package parser

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/blackstork-cli/parser/evaluation"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

func parseExecBlockDefConfig(
	blocksRegistry BlocksRegistry,
	execBlockDef *definitions.ExecBlockDef,
	configAttr *hclsyntax.Attribute,
	configBlock *hclsyntax.Block,
) (config evaluation.Configuration, diags diagnostics.Diag) {
	switch {
	case configAttr != nil && configBlock != nil:
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Both a config attribute and an inline config block are specified",
			Subject:  configBlock.DefRange().Ptr(),
			Context:  execBlockDef.Block.Body.Range().Ptr(),
		})
		return config, diags
	case configAttr != nil:
		// config attr referencing top-level config block
		cfg, diag := blocksRegistry.ResolveRefBase(configAttr.Expr, new(definitions.ConfigDef))
		if diags.Extend(diag) {
			return config, diags
		}

		cfgDef := cfg.(*definitions.ConfigDef)

		if !cfgDef.ApplicableTo(execBlockDef) {
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Mismatched configuration",
				Detail: fmt.Sprintf(
					"Referenced config block for `%s` blocks can't be applied to a `%s` block",
					cfgDef.ForKind(),
					execBlockDef.Key().FullName(),
				),
				Subject: configAttr.Range().Ptr(),
				Context: execBlockDef.Block.Body.Range().Ptr(),
			})
			return config, diags
		}

		config = &definitions.ConfigPtr{
			Cfg: cfgDef,
			Ptr: configAttr.AsHCLAttribute(),
		}
	case configBlock != nil:
		// anonymous config block
		config = &definitions.ConfigDef{
			Block: configBlock,
		}
	}
	return config, diags
}

// unique key (concatenation of type and labels).
func hclBlockKey(b *hclsyntax.Block) string {
	var sb strings.Builder
	length := len(b.Type) + len(b.Labels)
	for _, l := range b.Labels {
		length += len(l)
	}
	sb.Grow(length)
	sb.WriteString(b.Type)
	for _, l := range b.Labels {
		sb.WriteByte(0)
		sb.WriteString(l)
	}
	return sb.String()
}

func ParseStandaloneExecBlock(
	ctx context.Context,
	blocksRegistry BlocksRegistry,
	blockDef *definitions.ExecBlockDef,
) (definitions.ExecBlock, diagnostics.Diag) {
	switch blockDef.Kind() {
	case definitions.BlockKindContent:
		content, diags := parseContentBlock(ctx, blocksRegistry, blockDef, nil)
		return content, diags

	case definitions.BlockKindData:
		data, diags := ParseDataBlock(ctx, blocksRegistry, blockDef, nil)
		return data, diags

	case definitions.BlockKindPublish:
		publish, diags := parsePublishBlock(ctx, blocksRegistry, nil, blockDef, nil)
		return publish, diags

	case definitions.BlockKindFormat:
		format, diags := ParseFormatBlock(ctx, blocksRegistry, blockDef, nil)
		return format, diags
	}

	var diags diagnostics.Diag
	diags.Append(
		&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unknown exec block kind",
			Detail: fmt.Sprintf(
				"Unknown exec block kind encountered: %s",
				blockDef.Kind(),
			),
			Subject: blockDef.DefRange().Ptr(),
		},
	)

	return nil, diags
}
