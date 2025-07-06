package dataspec

import (
	"fmt"
	"slices"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
)

type Block struct {
	// Type + labels
	Header       []string
	HeaderRanges []hcl.Range

	Attrs  Attributes
	Blocks Blocks

	// Full range of the block, between {} (inclusive)
	ContentsRange hcl.Range
}

// Quickly create a new block without spec (no ranges will be assigned)
//
// Deprecated: use plugintest.NewTestDecoder instead
func NewBlock(headers []string, attrs map[string]cty.Value, blocks ...*Block) *Block {
	attrib := make(Attributes, len(attrs))
	for k, v := range attrs {
		attrib[k] = &Attr{
			Name:  k,
			Value: v,
		}
	}
	return &Block{
		Header:       headers,
		HeaderRanges: make([]hcl.Range, len(headers)),

		Attrs:  attrib,
		Blocks: blocks,
	}
}

func (b *Block) MissingItemRange() hcl.Range {
	r := b.ContentsRange
	r.End = r.Start
	return r
}

func (b *Block) DefRange() hcl.Range {
	return hcl.RangeBetween(b.HeaderRanges[0], b.HeaderRanges[len(b.HeaderRanges)-1])
}

func (b *Block) Range() hcl.Range {
	return hcl.RangeBetween(b.HeaderRanges[0], b.ContentsRange)
}

func (b *Block) HasAttr(name string) bool {
	if b == nil || b.Attrs == nil {
		return false
	}
	_, found := b.Attrs[name]
	return found
}

// Attempts to get attribute value, returns cty.NilVal if it's missing
func (b *Block) GetAttrVal(name string) cty.Value {
	if b == nil || b.Attrs == nil {
		return cty.NilVal
	}
	v, found := b.Attrs[name]
	if found && v != nil {
		return v.Value
	}
	return cty.NilVal
}

func (b *Block) GetAttrChecked(name string) (val *Attr, diags diagnostics.Diag) {
	if b == nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Block not found",
			Detail:   fmt.Sprintf("Attempted to get attribute %q on non-existent block", name),
		})
		return
	}
	val, found := b.Attrs[name]
	if !found || val == nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Attribute not found",
			Detail:   fmt.Sprintf("Attribute %q not found in block", name),
			Subject:  b.DefRange().Ptr(),
		})
		return
	}
	return
}

func (b Blocks) GetFirstMatching(header ...string) *Block {
	for _, block := range b {
		if slices.Equal(block.Header, header) {
			return block
		}
	}
	return nil
}
