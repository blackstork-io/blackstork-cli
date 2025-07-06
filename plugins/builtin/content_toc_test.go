package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
	"github.com/blackstork-io/fabric/internal/plugin/plugintest"
)

type TOCContentTestSuite struct {
	suite.Suite
	schema *plugin.ContentProvider
}

func TestTOCContentTestSuite(t *testing.T) {
	suite.Run(t, new(TOCContentTestSuite))
}

func (s *TOCContentTestSuite) SetupSuite() {
	s.schema = makeTOCContentProvider()
}

func (s *TOCContentTestSuite) TestSchema() {
	s.Require().NotNil(s.schema)
	s.Nil(s.schema.Config)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.ContentFunc)
}

func (s *TOCContentTestSuite) TestSimple() {
	val := cty.ObjectVal(map[string]cty.Value{})
	args := plugintest.ReencodeCTY(s.T(), s.schema.Args, val, nil)
	ctx := context.Background()
	titleMeta := plugindata.Map{
		"provider_name": plugindata.String("title"),
		"provider_plugin_name":   plugindata.String("blackstork/builtin"),
	}
	res, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,
		DataContext: plugindata.Map{
			"document": plugindata.Map{
				"content": plugindata.Map{
					"type": plugindata.String("section"),
					"children": plugindata.List{
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("# Header 1"),
							"meta":     titleMeta,
						},
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("Lorem ipsum dolor sit amet, consectetur adipiscing elit."),
						},
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("## Header 2"),
							"meta":     titleMeta,
						},
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("Vestibulum nec odio."),
						},
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("### Header 3"),
							"meta":     titleMeta,
						},
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("Integer sit amet."),
						},
					},
				},
			},
		},
	})
	s.Len(diags, 0, "no errors")
	s.Equal(strings.Join([]string{
		"- [Header 1](#header-1)",
		"  - [Header 2](#header-2)",
		"    - [Header 3](#header-3)",
	}, "\n")+"\n", res.Content)
}

func (s *TOCContentTestSuite) TestAdvanced() {
	val := cty.ObjectVal(map[string]cty.Value{
		"start_level": cty.NumberIntVal(1),
		"end_level":   cty.NumberIntVal(2),
		"ordered":     cty.True,
		"scope":       cty.StringVal("document"),
	})
	args := plugintest.ReencodeCTY(s.T(), s.schema.Args, val, nil)
	ctx := context.Background()
	titleMeta := plugindata.Map{
		"provider": plugindata.String("title"),
		"plugin":   plugindata.String("blackstork/builtin"),
	}
	res, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,

		DataContext: plugindata.Map{
			"document": plugindata.Map{
				"content": plugindata.Map{
					"type": plugindata.String("section"),
					"children": plugindata.List{
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("# Header 1"),
							"meta":     titleMeta,
						},
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("Lorem ipsum dolor sit amet, consectetur adipiscing elit."),
							"meta":     titleMeta,
						},
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("## Header 2"),
							"meta":     titleMeta,
						},
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("Vestibulum nec odio."),
						},
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("### Header 3"),
							"meta":     titleMeta,
						},
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("Integer sit amet."),
						},
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("## Header 4"),
							"meta":     titleMeta,
						},
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("Vestibulum nec odio."),
						},
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("## Header 5"),
							"meta":     titleMeta,
						},
						plugindata.Map{
							"type":     plugindata.String("element"),
							"markdown": plugindata.String("Vestibulum nec odio."),
						},
					},
				},
			},
		},
	})
	s.Len(diags, 0, "no errors")
	s.Equal(strings.Join([]string{
		"1. [Header 2](#header-2)",
		"  1. [Header 3](#header-3)",
		"2. [Header 4](#header-4)",
		"3. [Header 5](#header-5)",
	}, "\n")+"\n", res.Content)
}
