package parser

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/utils"
)

// Parses a section block
func parseSection(
	ctx context.Context,
	blocksRegistry BlocksRegistry,
	sectionDef *definitions.SectionDef,
	refHist *utils.RefHistory,
) (res *definitions.Section, diags diagnostics.Diag) {

	log := fabctx.GetLog(ctx)
	log.DebugContext(
		ctx, "Parsing a section",
		"name", sectionDef.Name(),
		"is_ref", sectionDef.IsRef(),
		"ref_hist", refHist.Size(),
	)

	// FIXME: WHY?
	//sectionDef.Once.Do(func() {
	// 	res, diags = db.parseSection(ctx, sectionDef)
	// 	if diags.HasErrors() {
	// 		return
	// 	}
	// 		sectionDef.ParseResult = res
	// 		sectionDef.Parsed = true
	// 	//})
	// 	if !sectionDef.Parsed {
	// 		if diags == nil {
	// 			diags.Append(diagnostics.RepeatedError)
	// 		}
	// 		return
	// 	}
	// 	res = sectionDef.ParseResult

	// Use a placeholder name if needed
	blockName := sectionDef.Name()
	if blockName == "" {
		blockName = makeNamePlaceholder()
	}

	res = &definitions.Section{
		Source:    sectionDef,
		BlockName: blockName,
	}

	// Parsing body
	body := sectionDef.Block.Body

	var diag diagnostics.Diag
	var validChildren []string

	isRef := sectionDef.IsRef()
	refBase, refBaseFound := body.Attributes[definitions.AttrRefBase]

	var targetSection *definitions.Section

	switch {
	case isRef && !refBaseFound:
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Ref block is missing `base` argument",
			Subject:  body.MissingItemRange().Ptr(),
			Context:  &body.SrcRange,
		})
		return nil, diags
	case !isRef && refBaseFound:
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Non-ref block contains `base` argument",
			Subject:  refBase.Range().Ptr(),
			Context:  &body.SrcRange,
		})
	case !isRef && !refBaseFound: // happy path, no ref
		validChildren = []string{
			definitions.BlockKindContent,
			definitions.BlockKindMeta,
			definitions.BlockKindSection,
			definitions.BlockKindVars,
			definitions.BlockKindDynamic,
		}
	case isRef && refBaseFound: // happy path, a correct ref block
		validChildren = []string{
			definitions.BlockKindMeta,
			definitions.BlockKindVars,
		}
		targetBlock, diag := parseRefBase(ctx, blocksRegistry, sectionDef, refBase.Expr, refHist)
		if diags.Extend(diag) {
			break
		}
		targetSection = targetBlock.(*definitions.Section)

		// Replace the name with base block name if not set in the definition
		if res.Source.Name() == "" {
			res.BlockName = targetSection.BlockName
		}
	}

	if diags.Extend(diag) {
		return nil, diags
	}

	if title := body.Attributes["title"]; title != nil {
		titleContent, diag := parseTitle(ctx, blocksRegistry, title)
		if !diag.Extend(diags) {
			res.Title = titleContent
		}
	} else if targetSection != nil && targetSection.Title != nil {
		res.Title = targetSection.Title
	}

	var origVars *definitions.Vars
	localVar := body.Attributes[definitions.AttrLocalVar]

	validChildrenSet := utils.SliceToSet(validChildren)

	for _, block := range body.Blocks {
		if !utils.Contains(validChildrenSet, block.Type) {
			diags.Append(definitions.NewNestingDiag(
				sectionDef.Block.Type,
				block,
				body,
				validChildren,
			))
			continue
		}
		switch block.Type {
		case definitions.BlockKindContent:
			contentDef, diag := definitions.DefineExecBlockDef(block, false)
			if diags.Extend(diag) {
				continue
			}
			content, diag := parseContentBlock(ctx, blocksRegistry, contentDef, refHist)
			if diags.Extend(diag) {
				continue
			}
			res.Content = append(res.Content, content)

		case definitions.BlockKindMeta:
			if res.Meta != nil {
				diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "More than one inline meta block defined",
					Detail:   "Only one inline meta block can be defined inside content block. Ignoring all meta blocks after the first one.",
					// Detail: fmt.Sprintf(
					// 	"Only one `meta` block allowed in `%s` and one is already defined at %s:%d",
					// 	sectionDef.Block.Type,
					// 	// origMeta.Filename,
					// 	// origMeta.Start.Line,
					// ),
					Subject: block.DefRange().Ptr(),
					Context: body.Range().Ptr(),
				})
				continue
			}
			var meta definitions.MetaBlock
			if diags.Extend(gohcl.DecodeBody(block.Body, nil, &meta)) {
				break
			}
			res.Meta = &meta

		case definitions.BlockKindVars:
			if origVars != nil {
				diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Vars block redefinition",
					Detail: fmt.Sprintf(
						"Only one inline `vars` block can be defined inside `%s` block. Ignoring all vars blocks after the first one",
						sectionDef.Block.Type,
						// block.DefRange().Filename,
						// block.DefRange().Start.Line,
					),
					Subject: block.DefRange().Ptr(),
					Context: body.Range().Ptr(),
				})
				break
			}

			origVars, diag = ParseVars(ctx, block, localVar)
			diags.Extend(diag)

		case definitions.BlockKindSection:
			subSectionDef, diag := definitions.DefineSectionDef(block, false)
			if diags.Extend(diag) {
				continue
			}
			subSection, diag := parseSection(ctx, blocksRegistry, subSectionDef, refHist)
			if diags.Extend(diag) {
				continue
			}
			res.Content = append(res.Content, subSection)
		case definitions.BlockKindDynamic:
			dynamic, diag := parseDynamic(ctx, blocksRegistry, block, refHist)
			if diags.Extend(diag) {
				continue
			}
			res.Content = append(res.Content, dynamic)
		}
	}

	if diags.Extend(diag) {
		return
	}

	res.Vars = &definitions.Vars{}
	if targetSection != nil {
		res.Vars.Extend(targetSection.Vars)
	}
	if origVars == nil {
		origVars, diag = ParseVars(ctx, nil, localVar)
		diags.Extend(diag)
	}
	res.Vars.Extend(origVars)

	// Collect `required_vars` and pop it to avoid validation errors when checking runner-specific attrs later
	var requiredVarsCombined []*hclsyntax.Attribute
	if reqVarsAttr, found := utils.Pop(body.Attributes, definitions.AttrRequiredVars); found {
		requiredVarsCombined = append(requiredVarsCombined, reqVarsAttr)
	}

	// Combine required vars from origin and target
	if targetSection != nil {
		requiredVarsCombined = append(requiredVarsCombined, targetSection.RequiredVarsCombined...)
	}
	res.RequiredVarsCombined = requiredVarsCombined

	// Collect `depends_on` and pop it to avoid validation errors when checking runner-specific attrs later
	var dependsOnCombined []*hclsyntax.Attribute
	if depAttr, ok := utils.Pop(body.Attributes, definitions.AttrDependsOn); ok {
		dependsOnCombined = append(dependsOnCombined, depAttr)
	}
	if targetSection != nil {
		dependsOnCombined = append(dependsOnCombined, targetSection.DependsOnCombined...)
	}
	res.DependsOnCombined = dependsOnCombined

	if origIsIncluded, found := body.Attributes[definitions.AttrIsIncluded]; found {
		res.IsIncluded = origIsIncluded
	} else if targetSection != nil && targetSection.IsIncluded != nil {
		res.IsIncluded = targetSection.IsIncluded
	}

	if res.Title == nil && targetSection != nil {
		res.Title = targetSection.Title
	}

	if res.Meta == nil && targetSection != nil {
		res.Meta = targetSection.Meta
	}

	return res, diags
}
