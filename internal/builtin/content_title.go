package builtin

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/utils"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

const (
	minAbsoluteTitleSize     = int64(0)
	maxAbsoluteTitleSize     = int64(5)
	defaultAbsoluteTitleSize = int64(0)
)

func makeTitleContentProvider() *plugin.ContentProvider {
	return &plugin.ContentProvider{
		ContentFunc: genTitleContent,
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "value",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
					Doc:         `Title content`,
					ExampleVal:  cty.StringVal("Vulnerability Report"),
				},
				{
					Name:        "absolute_size",
					Type:        cty.Number,
					Constraints: constraint.Integer,
					DefaultVal:  cty.NullVal(cty.Number),
					Doc: utils.Dedent(`
						Sets the absolute size of the title.
						If ` + "`null`" + ` – absoulute title size is determined from the document structure
					`),
				},
				{
					Name:        "relative_size",
					Type:        cty.Number,
					Constraints: constraint.Integer,
					DefaultVal:  cty.NumberIntVal(0),
					Doc: utils.Dedent(`
						Adjusts the absolute size of the title.
						The value (which may be negative) is added to the ` + "`absolute_size`" + ` to produce the final title size
					`),
				},
			},
		},
		Doc: utils.Dedent(`
			Produces a title.

			The title size after calculations must be in an interval [0; 5] inclusive, where 0
			corresponds to the largest size (` + "`<h1>`" + `) and 5 corresponds to (` + "`<h6>`" + `)
		`),
	}
}

func genTitleContent(
	ctx context.Context,
	params *plugin.ProvideContentParams,
) (*plugin.ContentResult, diagnostics.Diag) {
	value := params.Args.GetAttrVal("value")
	if value.IsNull() {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse arguments",
			Detail:   "value is required",
		}}
	}
	absoluteSize := params.Args.GetAttrVal("absolute_size")
	if absoluteSize.IsNull() {
		absoluteSize = cty.NumberIntVal(findDefaultTitleSize(params.DataContext) + 1)
	}
	relativeSize := params.Args.GetAttrVal("relative_size")

	titleSize, _ := absoluteSize.AsBigFloat().Int64()
	relationSize, _ := relativeSize.AsBigFloat().Int64()
	titleSize += relationSize
	if titleSize < minAbsoluteTitleSize {
		titleSize = minAbsoluteTitleSize
	}
	if titleSize < minAbsoluteTitleSize || titleSize > maxAbsoluteTitleSize {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse the arguments",
			Detail: fmt.Sprintf(
				"`absolute_size` value must be between %d and %d",
				minAbsoluteTitleSize,
				maxAbsoluteTitleSize,
			),
		}}
	}

	text, err := renderText(value.AsString(), params.DataContext)
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to render the value",
			Detail:   err.Error(),
		}}
	}
	// remove all newlines
	// text = strings.ReplaceAll(text, "\n", " ")
	// text = strings.Repeat("#", int(titleSize)+1) + " " + text
	return &plugin.ContentResult{
		Content: plugin.NewHeadingElement(text, titleSize, params.DataContext),
	}, nil
}

func findDefaultTitleSize(datactx plugindata.Map) int64 {
	document, section := parseScope(datactx)
	if section == nil {
		return defaultAbsoluteTitleSize
	}

	depth := findDepth(document, section.ID(), 1)
	if depth == 0 {
		return defaultAbsoluteTitleSize
	}
	size := defaultAbsoluteTitleSize + int64(depth)
	if size > maxAbsoluteTitleSize {
		return maxAbsoluteTitleSize
	}
	return size
}
