package builtin

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

func makeImageContentProvider() *plugin.ContentProvider {
	return &plugin.ContentProvider{
		ContentFunc: genImageContent,
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "src",
					Type:        cty.String,
					Constraints: constraint.RequiredMeaningful,
					ExampleVal:  cty.StringVal("https://example.com/img.png"),
				},
				{
					Name:       "alt",
					Type:       cty.String,
					ExampleVal: cty.StringVal("Text description of the image"),
					DefaultVal: cty.StringVal(""),
					// Not using empty string as DefaultVal here for semantical meaning
				},
			},
		},
		Doc: "Renders an image",
	}
}

func genImageContent(
	ctx context.Context,
	params *plugin.ProvideContentParams,
) (*plugin.ContentResult, diagnostics.Diag) {
	src := params.Args.GetAttrVal("src").AsString()
	alt := params.Args.GetAttrVal("alt").AsString()

	srcStr, err := renderText(src, params.DataContext)
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to render src value as a template",
			Detail:   err.Error(),
		}}
	}

	altStr, err := renderText(alt, params.DataContext)
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to render alt value as a template",
			Detail:   err.Error(),
		}}
	}

	// Make sure there are no line breaks in the values
	// srcStr = strings.TrimSpace(strings.ReplaceAll(srcStr, "\n", ""))
	// altStr = strings.TrimSpace(strings.ReplaceAll(altStr, "\n", ""))
	return &plugin.ContentResult{
		Content: plugin.NewImageElement(srcStr, altStr, params.DataContext),
	}, nil
}
