package builtin

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/constraint"
)

func makeCodeContentProvider() *plugin.ContentProvider {
	return &plugin.ContentProvider{
		ContentFunc: genCodeContent,
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "value",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
					ExampleVal:  cty.StringVal("Text to be formatted as a code block"),
				},
				{
					Name:       "language",
					Type:       cty.String,
					ExampleVal: cty.StringVal("json"),
					DefaultVal: cty.StringVal("text"),
					Doc:        `Specifiy the code language for syntax highlighting`,
				},
			},
		},
		Doc: "Formats text as a code snippet",
	}
}

func genCodeContent(ctx context.Context, params *plugin.ProvideContentParams) (*plugin.ContentResult, diagnostics.Diag) {
	value := params.Args.GetAttrVal("value").AsString()
	lang := params.Args.GetAttrVal("language").AsString()
	text, err := renderText(value, params.DataContext)
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to render code block as a template",
			Detail:   err.Error(),
		}}
	}
	//text = fmt.Sprintf("```%s\n%s\n```", lang.AsString(), text)

	return &plugin.ContentResult{
		Content: plugin.NewCodeElement(text, lang, params.DataContext),
	}, nil
}
