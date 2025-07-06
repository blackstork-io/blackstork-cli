package builtin

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/constraint"
)

func makeBlockQuoteContentProvider() *plugin.ContentProvider {
	return &plugin.ContentProvider{
		ContentFunc: genBlockQuoteContent,
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{{
				Name:        "value",
				Type:        cty.String,
				ExampleVal:  cty.StringVal("Text to be formatted as a quote"),
				Constraints: constraint.RequiredNonNull,
			}},
		},
		Doc: "Formats text as a blockquote",
	}
}

func genBlockQuoteContent(
	ctx context.Context,
	params *plugin.ProvideContentParams,
) (*plugin.ContentProviderResult, diagnostics.Diag) {
	value := params.Args.GetAttrVal("value")
	text, err := renderText(value.AsString(), params.DataContext)
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to render the value as a template",
			Detail:   err.Error(),
		}}
	}
	// text = "> " + strings.ReplaceAll(text, "\n", "\n> ")
	return &plugin.ContentProviderResult{
		Content: plugin.NewQuoteElement(text),
	}, nil
}
