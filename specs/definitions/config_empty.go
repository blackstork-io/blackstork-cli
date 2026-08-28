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
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/blackstork-cli/parser/evaluation"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

// ConfigEmpty stores the range of the original block
type ConfigEmpty struct {
	ExecBlockDef *ExecBlockDef
}

// Exists implements evaluation.Configuration.
func (c *ConfigEmpty) Exists() bool {
	return false
}

// ParseConfig implements Configuration.
func (c *ConfigEmpty) ParseConfig(ctx context.Context, spec *dataspec.RootSpec) (val *dataspec.Block, diags diagnostics.Diag) {
	block := c.ExecBlockDef.Block

	labels := make([]string, 1, len(block.Labels)+1)
	labels[0] = block.Type
	labels = append(labels, block.Labels...)
	labelRanges := make([]hcl.Range, 1, len(block.Labels)+1)
	labelRanges[0] = block.TypeRange
	labelRanges = append(labelRanges, block.LabelRanges...)

	if len(labels) >= 2 {
		// use the resolved name if it exists
		labels[1] = c.ExecBlockDef.Name()
	}

	emptyBody := hclsyntax.Block{
		Type:        "config",
		TypeRange:   block.TypeRange,
		Labels:      labels,
		LabelRanges: labelRanges,
		Body: &hclsyntax.Body{
			SrcRange: block.Body.MissingItemRange(),
			EndRange: block.Body.MissingItemRange(),
		},
		OpenBraceRange:  block.Body.MissingItemRange(),
		CloseBraceRange: block.Body.MissingItemRange(),
	}

	var diag diagnostics.Diag
	val, diag = dataspec.DecodeAndEvalBlock(ctx, &emptyBody, spec, nil)
	for _, d := range diag {
		d.Summary = fmt.Sprintf("Missing required configuration: %s", d.Summary)
	}
	return val, diagnostics.Diag(diag)
}

// Range implements Configuration.
func (c *ConfigEmpty) Range() hcl.Range {
	return c.ExecBlockDef.DefRange()
}

var _ evaluation.Configuration = (*ConfigEmpty)(nil)
