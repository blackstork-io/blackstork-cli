package plugintest

import (
	"context"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics/diagtest"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

// TestDecoder is a helper for testing block decoding.
type TestDecoder struct {
	t    *testing.T
	spec *dataspec.RootSpec

	block *TestBlock

	ctx     context.Context
	evalCtx *hcl.EvalContext
	dataCtx plugindata.Map
}

type TestBlock struct {
	block *hclwrite.Block
}

func NewTestBlock(typeName string, labels ...string) *TestBlock {
	return &TestBlock{
		block: hclwrite.NewBlock(typeName, labels),
	}
}

// SetAttr sets the attribute value.
func (tb *TestBlock) SetAttr(name string, value cty.Value) *TestBlock {
	tb.block.Body().SetAttributeValue(name, value)
	return tb
}

func (tb *TestBlock) AppendBody(body string) *TestBlock {
	f, diags := hclwrite.ParseConfig([]byte(body), "<tmp-file>", hcl.InitialPos)
	if diags.HasErrors() {
		panic(diags)
	}
	tb.block.Body().AppendUnstructuredTokens(f.Body().BuildTokens(nil))

	return tb
}

// AppendBlock appends a block to the body.
func (tb *TestBlock) AppendBlock(block *TestBlock) *TestBlock {
	tb.block.Body().AppendBlock(block.block)
	return tb
}

// SetHeaders sets the block type and labels.
func (tb *TestBlock) SetHeaders(typeName string, labels ...string) *TestBlock {
	tb.block.SetType(typeName)
	tb.block.SetLabels(labels)
	return tb
}

func (td *TestDecoder) AppendBlock(block *TestBlock) *TestDecoder {
	td.block.AppendBlock(block)
	return td
}

func (td *TestDecoder) AppendBody(body string) *TestDecoder {
	td.block.AppendBody(body)
	return td
}

func (td *TestDecoder) SetAttr(name string, value cty.Value) *TestDecoder {
	td.block.SetAttr(name, value)
	return td
}

func (td *TestDecoder) SetHeaders(typeName string, labels ...string) *TestDecoder {
	td.block.SetHeaders(typeName, labels...)
	return td
}

// NewTestDecoder creates a new TestDecoder.
// This is the preferred way to create the data for testing plugins.
func NewTestDecoder(t *testing.T, spec *dataspec.RootSpec) *TestDecoder {
	t.Helper()
	return &TestDecoder{
		block:   NewTestBlock("test_block"),
		t:       t,
		spec:    spec,
		dataCtx: plugindata.Map{},
		ctx:     context.Background(),
	}
}

// WithEvalCtx sets the evaluation context.
func (td *TestDecoder) WithEvalCtx(evalCtx *hcl.EvalContext) *TestDecoder {
	td.evalCtx = evalCtx
	return td
}

// WithDataCtx sets the data context.
func (td *TestDecoder) WithDataCtx(dataCtx plugindata.Map) *TestDecoder {
	td.dataCtx = dataCtx
	return td
}

// WithContext sets the context.
func (td *TestDecoder) WithContext(ctx context.Context) *TestDecoder {
	td.ctx = ctx
	return td
}

// Decode decodes the block and asserts diagnostics.
func (td *TestDecoder) Decode(asserts ...[]diagtest.Assert) (val *dataspec.Block) {
	td.t.Helper()
	val, fm, diags := td.DecodeDiagFiles()
	diagtest.Asserts.AssertMatch(asserts, td.t, diags, fm)
	return
}

// Decodes the block and returns diagnostics.
func (td *TestDecoder) DecodeDiag() (val *dataspec.Block, diags diagnostics.Diag) {
	td.t.Helper()
	val, _, diags = td.DecodeDiagFiles()
	return
}

func (td *TestDecoder) DecodeDiagFiles() (val *dataspec.Block, fm map[string]*hcl.File, diags diagnostics.Diag) {
	td.t.Helper()
	file := hclwrite.NewFile()
	file.Body().AppendBlock(td.block.block)
	data := hclwrite.Format(file.Bytes())
	filename := getFileName()
	fm = map[string]*hcl.File{
		filename: {
			Body:  nil,
			Bytes: data,
		},
	}

	f, diag := hclsyntax.ParseConfig(data, filename, hcl.InitialPos)
	if diags.Extend(diag) {
		return
	}
	fm[filename] = f
	if td.evalCtx == nil {
		td.evalCtx = fabctx.GetEvalContext(td.ctx)
	}
	val, dgs := dataspec.DecodeAndEvalBlock(td.ctx, f.Body.(*hclsyntax.Body).Blocks[0], td.spec, td.dataCtx)
	if diags.Extend(dgs) {
		return
	}
	return
}
