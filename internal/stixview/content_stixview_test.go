package stixview

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/blackstork-io/fabric/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/plugindata"
	"github.com/blackstork-io/fabric/plugin/plugintest"
)

type StixViewTestSuite struct {
	suite.Suite
	schema *plugin.ContentProvider
}

func TestStixViewTestSuite(t *testing.T) {
	suite.Run(t, new(StixViewTestSuite))
}

func (s *StixViewTestSuite) SetupTest() {
	s.schema = makeStixViewContentProvider()
}

func (s *StixViewTestSuite) TestSchema() {
	s.Require().NotNil(s.schema)
	s.Nil(s.schema.Config)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.ContentFunc)
}

func (s *StixViewTestSuite) TestGistID() {
	dataCtx := plugindata.Map{}
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		gist_id = "123"
	`, dataCtx, diagtest.Asserts{})

	res, diags := s.schema.ContentFunc(context.Background(), &plugin.ProvideContentParams{
		Args:        args,
		DataContext: dataCtx,
	})
	s.Empty(diags)
	s.Equal(strings.Join([]string{
		`<script src="https://unpkg.com/stixview/dist/stixview.bundle.js" type="text/javascript"></script>`,
		`<div data-stix-gist-id="123">`,
		`</div>`,
		``,
	// }, "\n"), mdprint.PrintString(res.Content))
	}, "\n"), res.Content)
}

func (s *StixViewTestSuite) TestStixURL() {
	dataCtx := plugindata.Map{}
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		stix_url = "https://example.com/stix.json"
	`, dataCtx, diagtest.Asserts{})

	res, diags := s.schema.ContentFunc(context.Background(), &plugin.ProvideContentParams{
		Args:        args,
		DataContext: dataCtx,
	})

	s.Empty(diags)
	s.Equal(strings.Join([]string{
		`<script src="https://unpkg.com/stixview/dist/stixview.bundle.js" type="text/javascript"></script>`,
		`<div data-stix-url="https://example.com/stix.json">`,
		`</div>`,
		``,
	//}, "\n"), mdprint.PrintString(res.Content))
	}, "\n"), res.Content)
}

func (s *StixViewTestSuite) TestAllArgs() {
	dataCtx := plugindata.Map{}
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		gist_id = "123"
		caption = "test caption"
		show_footer = true
		show_sidebar = true
		show_tlp_as_tags = true
		show_marking_nodes = true
		show_labels = true
		show_idrefs = true
		width = 400
		height = 300

	`, dataCtx, diagtest.Asserts{})

	res, diags := s.schema.ContentFunc(context.Background(), &plugin.ProvideContentParams{
		Args:        args,
		DataContext: dataCtx,
	})

	s.Empty(diags)
	s.Equal(strings.Join([]string{
		`<script src="https://unpkg.com/stixview/dist/stixview.bundle.js" type="text/javascript"></script>`,
		`<div data-stix-gist-id="123" data-show-sidebar="true" data-show-footer="true" data-show-tlp-as-tags="true" data-caption="test caption" data-show-marking-nodes="true" data-show-labels="true" data-show-idrefs="true" data-graph-width="400" data-graph-height="300">`,
		`</div>`,
		``,
	//}, "\n"), mdprint.PrintString(res.Content))
	}, "\n"), res.Content)
}

func (s *StixViewTestSuite) TestDataCtx() {
	dataCtx := plugindata.Map{}
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		objects = [
			{"key" = "value"}
		]

	`, dataCtx, diagtest.Asserts{})

	res, diags := s.schema.ContentFunc(context.Background(), &plugin.ProvideContentParams{
		Args:        args,
		DataContext: dataCtx,
	})
	s.Empty(diags)
	s.Contains(res.Content, `<script src="https://unpkg.com/stixview/dist/stixview.bundle.js" type="text/javascript"></script>`)
	s.Contains(res.Content, `<div id="graph-`)
	s.Contains(res.Content, `window.stixview.init(`)
	s.Contains(res.Content, `"objects": [{"key":"value"}]`)
}
