package builtin

import (
	"context"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/plugindata"
	"github.com/blackstork-io/fabric/plugin/plugintest"
)

type TextTestSuite struct {
	suite.Suite
	schema *plugin.ContentProvider
}

func TestTextSuite(t *testing.T) {
	suite.Run(t, &TextTestSuite{})
}

func (s *TextTestSuite) SetupSuite() {
	s.schema = makeTextContentProvider()
}

func (s *TextTestSuite) TestSchema() {
	s.Nil(s.schema.Config)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.ContentFunc)
}

func (s *TextTestSuite) TestMissingText() {
	val := cty.ObjectVal(map[string]cty.Value{
		"value": cty.NullVal(cty.String),
	})
	plugintest.ReencodeCTY(s.T(), s.schema.Args, val, diagtest.Asserts{{
		diagtest.IsError,
		diagtest.SummaryContains("Attribute must be non-null"),
	}})
}

func (s *TextTestSuite) TestBasic() {
	val := cty.ObjectVal(map[string]cty.Value{
		"value": cty.StringVal("Hello {{.name}}!"),
	})
	args := plugintest.ReencodeCTY(s.T(), s.schema.Args, val, nil)

	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,
		DataContext: plugindata.Map{
			"name": plugindata.String("World"),
		},
	})
	s.Empty(diags)
	// s.Equal("Hello World!", result.Content)
	s.Equal("Hello World!", result.Content)
}

func (s *TextTestSuite) TestNoTemplate() {
	val := cty.ObjectVal(map[string]cty.Value{
		"value": cty.StringVal("Hello World!"),
	})
	args := plugintest.ReencodeCTY(s.T(), s.schema.Args, val, nil)
	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args:        args,
		DataContext: nil,
	})
	s.Empty(diags)
	s.Equal("Hello World!", result.Content)
}

func (s *TextTestSuite) TestCallInvalidTemplate() {
	val := cty.ObjectVal(map[string]cty.Value{
		"value": cty.StringVal("Hello {{.name}!"),
	})
	args := plugintest.ReencodeCTY(s.T(), s.schema.Args, val, nil)

	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,
		DataContext: plugindata.Map{
			"name": plugindata.String("World"),
		},
	})
	s.Nil(result)
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to render text",
		Detail:   "failed to parse text template: template: text:1: bad character U+007D '}'",
	}}, diags)
}

func (s *TextTestSuite) TestSprigTemplate() {
	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: &dataspec.Block{
			Attrs: dataspec.Attributes{
				"value": &dataspec.Attr{
					Name:  "value",
					Value: cty.StringVal("Hello {{.name | upper}}!"),
				},
			},
		},
		DataContext: plugindata.Map{
			"name": plugindata.String("World"),
		},
	})
	s.Empty(diags)
	s.Equal("Hello WORLD!", result.Content)
}
