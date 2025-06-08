package builtin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/blackstork-io/fabric/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/plugindata"
	"github.com/blackstork-io/fabric/plugin/plugintest"
)

type BlockQuoteTestSuite struct {
	suite.Suite
	schema *plugin.ContentProvider
}

func TestBlockQuoteSuite(t *testing.T) {
	suite.Run(t, &BlockQuoteTestSuite{})
}

func (s *BlockQuoteTestSuite) SetupSuite() {
	s.schema = makeBlockQuoteContentProvider()
}

func (s *BlockQuoteTestSuite) TestSchema() {
	s.Nil(s.schema.Config)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.ContentFunc)
}

func (s *BlockQuoteTestSuite) TestMissingText() {
	plugintest.DecodeAndAssert(s.T(), s.schema.Args, ``, nil, diagtest.Asserts{
		{
			diagtest.IsError,
			diagtest.DetailContains(`The attribute "value" is required`),
		},
	})
}

func (s *BlockQuoteTestSuite) TestNullText() {
	plugintest.DecodeAndAssert(s.T(), s.schema.Args, `value = null`, nil, diagtest.Asserts{
		{
			diagtest.IsError,
			diagtest.SummaryContains(`Attribute must be non-null`),
		},
	})
}

func (s *BlockQuoteTestSuite) TestCallBlockquote() {
	ctx := context.Background()
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		value = "Hello {{.name}}!"
	`, nil, nil)
	content, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,
		DataContext: plugindata.Map{
			"name": plugindata.String("World"),
		},
	})
	s.Empty(diags)
	s.Equal(
		plugindata.String("> Hello World!"),
		content.Content.AsData()["markdown"],
	)
}

func (s *BlockQuoteTestSuite) TestCallBlockquoteMultiline() {
	ctx := context.Background()
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		value = "Hello\n{{.name}}\nfor you!"
	`, nil, nil)
	content, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,
		DataContext: plugindata.Map{
			"name": plugindata.String("World"),
		},
	})
	s.Empty(diags)
	s.Equal(
		plugindata.String("> Hello\n> World\n> for you!"),
		content.Content.AsData()["markdown"],
	)
}

func (s *BlockQuoteTestSuite) TestCallBlockquoteMultilineDoubleNewline() {
	ctx := context.Background()
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		value = "Hello\n{{.name}}\n\nfor you!"
	`, nil, nil)
	content, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,
		DataContext: plugindata.Map{
			"name": plugindata.String("World"),
		},
	})
	s.Empty(diags)
	s.Equal(
		plugindata.String("> Hello\n> World\n> \n> for you!"),
		content.Content.AsData()["markdown"],
	)
}
