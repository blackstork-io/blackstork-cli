package basic

import (
	"context"
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/constraint"
)

// makeGreetingContentProvider creates a new content provider that prints out a greeting message
func makeGreetingContentProvider() *plugin.ContentProvider {
	return &plugin.ContentProvider{
		// Config is optional, in this case we don't need it
		// We only define the schema for the arguments
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{{
				Name:        "name",
				Constraints: constraint.RequiredMeaningful,
				Doc:         `Name of the user`,
				ExampleVal:  cty.StringVal("John"),
				Type:        cty.String,
			}},
		},
		// Optional: We can also define the schema for the config
		ContentFunc: renderGreetingMessage,
	}
}

func renderGreetingMessage(
	ctx context.Context,
	params *plugin.ProvideContentParams,
) (*plugin.ContentResult, diagnostics.Diag) {
	// We specified that the "name" attribute is RequiredMeaningful, so we can safely assume
	// that it exists, non-null and non-empty, with whitespace trimmed
	name := params.Args.GetAttrVal("name").AsString()
	text := fmt.Sprintf("Hello, %s!", name)

	return &plugin.ContentResult{
		Content: plugin.NewTextElement(text, params.DataContext),
	}, nil
}
