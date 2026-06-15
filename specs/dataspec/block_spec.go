// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package dataspec

import (
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
)

type NameMatcher interface {
	Match(ty string, labels []string) bool
}

type HeadersSpec []NameMatcher

type ExactMatcher []string

var _ NameMatcher = ExactMatcher(nil)

func (m ExactMatcher) Match(ty string, labels []string) bool {
	if len(m) != len(labels)+1 {
		return false
	}
	if m[0] != ty {
		return false
	}
	for i, match := range m[1:] {
		if match != labels[i] {
			return false
		}
	}
	return true
}

func (hs HeadersSpec) Match(ty string, labels []string) bool {
	for _, m := range hs {
		if !m.Match(ty, labels) {
			return false
		}
	}
	return true
}

func (hs HeadersSpec) AsDocLabels() (name string, labels []string) {
	for _, m := range hs {
		if em, ok := m.(ExactMatcher); ok {
			return em[0], em[1:]
		}
	}
	return "<any-block>", nil
}

type BlockSpec struct {
	Header     HeadersSpec
	Required   bool
	Repeatable bool

	Doc string

	Blocks []*BlockSpec
	Attrs  []*AttrSpec

	AllowUnspecifiedBlocks     bool
	AllowUnspecifiedAttributes bool
}

func (b *BlockSpec) WriteBlockDoc(w *hclwrite.Body) {
	trimmedDoc := strings.Trim(b.Doc, "\n ")
	tokens := comment(nil, trimmedDoc)
	if len(tokens) != 0 {
		tokens = appendCommentNewLine(tokens)
	}
	var comm string
	if b.Required {
		comm = "Required"
	} else {
		comm = "Optional"
	}
	if b.Repeatable {
		comm += ", can be repeated"
	}
	tokens = comment(tokens, comm)

	w.AppendUnstructuredTokens(tokens)
	block := w.AppendNewBlock(b.Header.AsDocLabels())
	b.WriteBodyDoc(block.Body())
}

func (b *BlockSpec) WriteBodyDoc(w *hclwrite.Body) {
	if len(b.Blocks) != 0 {
		b.Blocks[0].WriteBlockDoc(w)
		for _, subBlock := range b.Blocks[1:] {
			w.AppendNewline()
			subBlock.WriteBlockDoc(w)
		}
	}
	if len(b.Attrs) != 0 {
		if len(b.Blocks) != 0 {
			w.AppendNewline()
			w.AppendNewline()
		}
		b.Attrs[0].WriteDoc(w)
		for _, subAttr := range b.Attrs[1:] {
			w.AppendNewline()
			subAttr.WriteDoc(w)
		}
	}
}

func (b *BlockSpec) ValidateSpec() (errs diagnostics.Diag) {
	for _, subB := range b.Blocks {
		errs.Extend(subB.ValidateSpec())
	}
	for _, attr := range b.Attrs {
		errs.Extend(attr.ValidateSpec())
	}
	return errs
}
