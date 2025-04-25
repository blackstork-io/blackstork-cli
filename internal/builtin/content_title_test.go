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
	"github.com/blackstork-io/fabric/plugin/plugindata"
	"github.com/blackstork-io/fabric/plugin/plugintest"
)

type TitleTestSuite struct {
	suite.Suite
	schema *plugin.ContentProvider
}

func TestTitleSuite(t *testing.T) {
	suite.Run(t, &TitleTestSuite{})
}

func (s *TitleTestSuite) SetupSuite() {
	s.schema = makeTitleContentProvider()
}

func (s *TitleTestSuite) TestSchema() {
	s.Nil(s.schema.Config)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.ContentFunc)
}

func (s *TitleTestSuite) TestMissingValue() {
	val := cty.ObjectVal(map[string]cty.Value{
		"value":         cty.NullVal(cty.String),
		"absolute_size": cty.NullVal(cty.Number),
		"relative_size": cty.NullVal(cty.Number),
	})
	plugintest.ReencodeCTY(s.T(), s.schema.Args, val, diagtest.Asserts{{
		diagtest.IsError,
		diagtest.SummaryContains("Attribute must be non-null"),
	}})
}

func (s *TitleTestSuite) TestTDefault() {
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
	s.Equal("## Hello World!", result.Content)
}

func (s *TitleTestSuite) TestWithTextMultiline() {
	val := cty.ObjectVal(map[string]cty.Value{
		"value": cty.StringVal("Hello\n{{.name}}\nfor you!"),
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
	s.Equal("## Hello World for you!", result.Content)
}

func (s *TitleTestSuite) TestWithSize() {
	val := cty.ObjectVal(map[string]cty.Value{
		"value":         cty.StringVal("Hello {{.name}}!"),
		"absolute_size": cty.NumberIntVal(2),
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
	s.Equal("### Hello World!", result.Content)
}

func (s *TitleTestSuite) TestWithSizeTooBig() {
	val := cty.ObjectVal(map[string]cty.Value{
		"value":         cty.StringVal("Hello {{.name}}!"),
		"absolute_size": cty.NumberIntVal(7),
	})
	args := plugintest.ReencodeCTY(s.T(), s.schema.Args, val, nil)
	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,
		DataContext: plugindata.Map{
			"name": plugindata.String("World"),
		},
	})
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to parse arguments",
		Detail:   "absolute_size must be between 0 and 5",
	}}, diags)
	s.Nil(result)
}
