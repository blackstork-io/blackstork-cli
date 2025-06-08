package parser

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/utils"
)

func (db *DefinedBlocks) ParseTitle(
	ctx context.Context,
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
		return
	}
	contentBlock, diag := db.ParseContentBlock(ctx, def, nil)
	if diags.Extend(diag) {
		return
	}
	return contentBlock, diags
}
