// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

// Documentable wrapper types form
package dataspec

import (
	"bytes"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
)

type (
	Blocks     []*Block
	Attributes map[string]*Attr
)

// RenderDoc renders the block documentation for spec.
func RenderDoc(spec *RootSpec, blockName string, labels ...string) string {
	// Special-casing the first line generation:
	// config "data" "csv" { -> config data csv {

	if strings.Contains(blockName, " ") {
		return "<error: block name contains spaces>"
	}

	f := hclwrite.NewEmptyFile()
	spec.BlockSpec().WriteBodyDoc(f.Body().AppendNewBlock(blockName, labels).Body())
	doc := hclwrite.Format(f.Bytes())
	blockBodyStart := bytes.IndexByte(doc, '{')
	if blockBodyStart == -1 {
		return "<error: no block body generated>"
	}

	header := formatHeader(blockName, labels)
	header = append(header, " "...)
	newStart := blockBodyStart - len(header)
	if newStart >= 0 {
		copy(doc[newStart:], header)
		doc = doc[newStart:]
	} else {
		doc = append(header, doc[blockBodyStart:]...)
	}

	return string(doc)
}

func (b Blocks) GetFirstMatching(header ...string) *Block {
	for _, block := range b {
		if slices.Equal(block.Header, header) {
			return block
		}
	}
	return nil
}
