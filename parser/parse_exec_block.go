package parser

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/parser/evaluation"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
)

// // Evaluates a defined exec block
// func (db *DefinedBlocks) ParseExecBlock(
// 	ctx context.Context,
// 	block *definitions.ExecBlockDef,
// ) (res definitions.ExecBlock, diags diagnostics.Diag) {
// 	if circularRefDetector.Check(block) {
// 		// This produces a bit of an incorrect error and shouldn't trigger in normal operation
// 		// but I re-check for the circular refs here out of abundance of caution:
// 		// deadlocks or infinite loops may occur, and are hard to debug
// 		diags.Append(&hcl.Diagnostic{
// 			Severity: hcl.DiagError,
// 			Summary:  "Circular reference detected",
// 			Detail:   "Looped back to this block through reference chain:",
// 			Subject:  block.DefRange().Ptr(),
// 			Extra:    diagnostics.NewTracebackExtra(),
// 		})
// 		return
// 	}
// 	// FIXME: Why?
// 	//block.Once.Do(func() {
//
// 	switch block.Kind() {
// 	case definitions.BlockKindContent:
// 		res, diags = db.parseContentBlock(ctx, block)
// 	case definitions.BlockKindData:
// 		res, diags = db.parseDataBlock(ctx, block)
// 	case definitions.BlockKindPublish:
// 		res, diags = db.parsePublishBlock(ctx, block)
// 	case definitions.BlockKindFormat:
// 		res, diags = db.parseFormatBlock(ctx, block)
// 	default:
// 		diags.Add(
// 			"Unknown block type encountered",
// 			fmt.Sprintf("Unknown block type `%s` encountered", block.Type),
// 		)
// 	}
// 	if diags.HasErrors() {
// 		return
// 	}
// 	//})
// // 	if !block.Parsed {
// // 		if diags == nil {
// // 			diags.Append(diagnostics.RepeatedError)
// // 		}
// // 		return
// // 	}
// 	return
// }

func (db *DefinedBlocks) parseExecBlockDefConfig(
	execBlockDef *definitions.ExecBlockDef,
	configAttr *hclsyntax.Attribute,
	configBlock *hclsyntax.Block,
) (config evaluation.Configuration, diags diagnostics.Diag) {
	switch {
	case configAttr != nil && configBlock != nil:
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Both a config attribute and an inline config block are specified",
			Subject:  configBlock.DefRange().Ptr(),
			Context:  execBlockDef.Block.Body.Range().Ptr(),
		})
		return
	case configAttr != nil:
		// config attr referencing top-level config block
		cfg, diag := ResolveWithDefined[*definitions.ConfigDef](db, configAttr.Expr)
		if diags.Extend(diag) {
			return
		}
		if !cfg.ApplicableTo(execBlockDef) {
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Mismatched configuration",
				Detail:   "The configuration is not applicable for used block type",
				Subject:  configAttr.Range().Ptr(),
				Context:  execBlockDef.Block.Body.Range().Ptr(),
			})
			return
		}

		config = &definitions.ConfigPtr{
			Cfg: cfg,
			Ptr: configAttr.AsHCLAttribute(),
		}
	case configBlock != nil:
		// anonymous config block
		config = &definitions.ConfigDef{
			Block: configBlock,
		}
	}
	return
}

// unique key (concatenation of type and labels).
func hclBlockKey(b *hclsyntax.Block) string {
	var sb strings.Builder
	length := len(b.Type) + len(b.Labels)
	for _, l := range b.Labels {
		length += len(l)
	}
	sb.Grow(length)
	sb.WriteString(b.Type)
	for _, l := range b.Labels {
		sb.WriteByte(0)
		sb.WriteString(l)
	}
	return sb.String()
}
