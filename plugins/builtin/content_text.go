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

func makeTextContentProvider() *plugin.ContentProvider {
	return &plugin.ContentProvider{
		ContentFunc: genTextContent,
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "value",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
					ExampleVal:  cty.StringVal("Hello world!"),
					Doc:         `Text value rendered as a Go template`,
				},
			},
		},
		Doc: `Renders text block`,
	}
}

func genTextContent(
	ctx context.Context,
	params *plugin.ProvideContentParams,
) (*plugin.ContentProviderResult, diagnostics.Diag) {
	value := params.Args.GetAttrVal("value")
	if value.IsNull() {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse the arguments",
			Detail:   "`value` must be provided",
		}}
	}

	text, err := renderText(value.AsString(), params.DataContext)
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to render text block",
			Detail:   err.Error(),
		}}
	}
	return &plugin.ContentProviderResult{
		Content: plugin.NewTextElement(text),
	}, nil
}
