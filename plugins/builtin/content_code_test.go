package builtin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
	"github.com/blackstork-io/fabric/internal/plugin/plugintest"
)

type CodeTestSuite struct {
	suite.Suite
	schema *plugin.ContentProvider
}

func TestCodeSuite(t *testing.T) {
	suite.Run(t, &CodeTestSuite{})
}

func (s *CodeTestSuite) SetupSuite() {
	s.schema = makeCodeContentProvider()
}

func (s *CodeTestSuite) TestSchema() {
	s.Nil(s.schema.Config)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.ContentFunc)
}

func (s *CodeTestSuite) TestMissingValue() {
	val := cty.ObjectVal(map[string]cty.Value{
		"value":    cty.NullVal(cty.String),
		"language": cty.NullVal(cty.String),
	})
	plugintest.ReencodeCTY(s.T(), s.schema.Args, val, diagtest.Asserts{{
		diagtest.SummaryContains("Attribute must be non-null"),
	}})
}

func (s *CodeTestSuite) TestCallCodeDefault() {
	ctx := context.Background()

	args := plugintest.NewTestDecoder(s.T(), s.schema.Args).
		SetAttr("value", cty.StringVal("Hello {{.name}}!")).
		Decode()

	content, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,
		DataContext: plugindata.Map{
			"name": plugindata.String("World"),
		},
	})
	s.Empty(diags)
	s.Equal(
		plugindata.String("```\nHello World!\n```"),
		content.Content.AsData().(plugindata.Map)["markdown"],
	)
}

func (s *CodeTestSuite) TestCallCodeWithLanguage() {
	ctx := context.Background()
	val := cty.ObjectVal(map[string]cty.Value{
		"value":    cty.StringVal(`{"hello": "{{.name}}"}`),
		"language": cty.StringVal("json"),
	})
	args := plugintest.ReencodeCTY(s.T(), s.schema.Args, val, nil)

	content, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,
		DataContext: plugindata.Map{
			"name": plugindata.String("world"),
		},
	})
	s.Empty(diags)
	s.Equal(
		plugindata.String("```json\n{\"hello\": \"world\"}\n```"),
		content.Content.AsData().(plugindata.Map)["markdown"],
	)
}
