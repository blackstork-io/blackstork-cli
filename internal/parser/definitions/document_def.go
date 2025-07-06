package definitions

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/utils/encapsulator"
)

type DocumentDef struct {
	Block *hclsyntax.Block
	Name  string
	Meta  *MetaBlock
}

func (d *DocumentDef) Kind() string {
	return BlockKindDocument
}

var _ BlockDef = (*DocumentDef)(nil)

func (d *DocumentDef) GetHCLBlock() *hclsyntax.Block {
	return d.Block
}

var ctyDocumentType = encapsulator.NewEncoder[DocumentDef]("document", nil)

func (d *DocumentDef) CtyType() cty.Type {
	return ctyDocumentType.CtyType()
}

func DefineDocumentDef(block *hclsyntax.Block) (doc *DocumentDef, diags diagnostics.Diag) {
	diags.Append(validateBlockName(block, 0, true))
	diags.Append(validateLabelsLength(block, 1, "document_name"))
	if diags.HasErrors() {
		return
	}

	if block.Labels[0] == BlockTypeRef {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid document definition",
			Detail:   "Documents can't be marked as references",
			Subject:  &block.LabelRanges[0],
			Context:  block.DefRange().Ptr(),
		})
	}

	doc = &DocumentDef{
		Block: block,
		Name:  block.Labels[0],
	}
	return
}
