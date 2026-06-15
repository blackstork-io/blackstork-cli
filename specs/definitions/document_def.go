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

type DocumentDef struct {
	block *hclsyntax.Block
	Name  string
	// Meta  *MetaBlock
}

func (d *DocumentDef) Kind() string {
	return BlockKindDocument
}

var _ BlockDef = (*DocumentDef)(nil)

func (d *DocumentDef) GetHCLBlock() *hclsyntax.Block {
	return d.block
}

var ctyDocumentType = encapsulator.NewEncoder[DocumentDef]("document", nil)

func (d *DocumentDef) CtyType() cty.Type {
	return ctyDocumentType.CtyType()
}

func DefineDocumentDef(block *hclsyntax.Block) (doc *DocumentDef, diags diagnostics.Diag) {
	diags.Append(validateBlockName(block, 0, true))
	diags.Append(validateLabelsLength(block, 1, "document_name"))
	if diags.HasErrors() {
		return doc, diags
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
		block: block,
		Name:  block.Labels[0],
	}
	return doc, diags
}
