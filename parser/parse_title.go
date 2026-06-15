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

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

func parseTitle(
	ctx context.Context,
	blocksRegistry BlocksRegistry,
	title *hclsyntax.Attribute,
) (res *definitions.ContentBlock, diags diagnostics.Diag) {
	const pluginName = "title"

	value := *title
	value.Name = "value"

	relativeSize := *title
	relativeSize.Name = "relative_size"
	relativeSize.Expr = &hclsyntax.LiteralValueExpr{
		Val:      cty.NumberIntVal(0),
		SrcRange: title.Expr.Range(),
	}

	block := &hclsyntax.Block{
		Type:        definitions.BlockKindContent,
		TypeRange:   title.NameRange,
		Labels:      []string{pluginName},
		LabelRanges: []hcl.Range{title.NameRange},
		Body: &hclsyntax.Body{
			Attributes: hclsyntax.Attributes{
				"value":         &value,
				"relative_size": &relativeSize,
			},
			SrcRange: title.SrcRange,
			EndRange: utils.RangeEnd(title.Expr.Range()),
		},
		OpenBraceRange:  utils.RangeStart(title.NameRange),
		CloseBraceRange: utils.RangeEnd(title.Expr.Range()),
	}

	def, diag := definitions.DefineExecBlockDef(block, false)
	if diags.Extend(diag) {
		return res, diags
	}
	contentBlock, diag := parseContentBlock(ctx, blocksRegistry, def, nil)
	if diags.Extend(diag) {
		return res, diags
	}
	return contentBlock, diags
}
