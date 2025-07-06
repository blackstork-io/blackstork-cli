// Documentable wrapper types form
package dataspec

import (
	"bytes"
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
	newStart := blockBodyStart - len(header)
	if newStart >= 0 {
		copy(doc[newStart:], header)
		doc = doc[newStart:]
	} else {
		doc = append(header, doc[blockBodyStart:]...)
	}

	return string(doc)
}
