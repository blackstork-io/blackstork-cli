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
	"github.com/google/uuid"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

type Ctyable interface {
	CtyType() cty.Type
}

func makeNamePlaceholder() string {
	id := uuid.New()
	return id.String()[:8]
}

// cloneHCLBlock creates a parsing-owned AST. Parsers remove common attributes
// and blocks before runner evaluation, so they must not operate on definitions
// stored in the shared block registry.
func cloneHCLBlock(block *hclsyntax.Block) *hclsyntax.Block {
	if block == nil {
		return nil
	}
	cloned := *block
	cloned.Labels = append([]string(nil), block.Labels...)
	cloned.LabelRanges = append(cloned.LabelRanges[:0:0], block.LabelRanges...)
	cloned.Body = cloneHCLBody(block.Body)
	return &cloned
}

func cloneHCLBody(body *hclsyntax.Body) *hclsyntax.Body {
	if body == nil {
		return nil
	}
	cloned := *body
	cloned.Attributes = make(hclsyntax.Attributes, len(body.Attributes))
	for name, attr := range body.Attributes {
		clonedAttr := *attr
		cloned.Attributes[name] = &clonedAttr
	}
	cloned.Blocks = make(hclsyntax.Blocks, len(body.Blocks))
	for i, block := range body.Blocks {
		cloned.Blocks[i] = cloneHCLBlock(block)
	}
	return &cloned
}

func cloneExecBlockDef(def *definitions.ExecBlockDef) *definitions.ExecBlockDef {
	return &definitions.ExecBlockDef{Block: cloneHCLBlock(def.Block)}
}

func cloneSectionDef(def *definitions.SectionDef) *definitions.SectionDef {
	return &definitions.SectionDef{Block: cloneHCLBlock(def.Block)}
}
