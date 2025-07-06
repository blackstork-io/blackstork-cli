package misp_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/misp"
	"github.com/blackstork-io/fabric/internal/misp/client"
	mocks "github.com/blackstork-io/fabric/mocks/internalpkg/misp"
	"github.com/blackstork-io/fabric/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
	"github.com/blackstork-io/fabric/internal/plugin/plugintest"
)

type MispPublishEventReportsTestSuite struct {
	suite.Suite
	plugin *plugin.Schema
	cli    *mocks.Client
}

func TestGithubublishGistSuite(t *testing.T) {
	suite.Run(t, &MispPublishEventReportsTestSuite{})
}

func (s *MispPublishEventReportsTestSuite) SetupSuite() {
	s.plugin = misp.Plugin("1.2.3", func(cfg *dataspec.Block) misp.Client {
		return s.cli
	})
}

func (s *MispPublishEventReportsTestSuite) SetupTest() {
	s.cli = &mocks.Client{}
}

func (s *MispPublishEventReportsTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *MispPublishEventReportsTestSuite) PublisherName() string {
	return "misp_event_reports"
}

func (s *MispPublishEventReportsTestSuite) Publisher() *plugin.Publisher {
	return s.plugin.Publishers[s.PublisherName()]
}

func (s *MispPublishEventReportsTestSuite) TestSchema() {
	schema := s.plugin.Publishers["misp_event_reports"]
	s.Require().NotNil(schema)
	s.NotNil(schema.Config)
	s.NotNil(schema.Args)
	s.NotNil(schema.PublishFunc)
}

func (s *MispPublishEventReportsTestSuite) TestBasic() {
	uuid := uuid.New().String()
	s.cli.On("AddEventReport", mock.Anything, mock.Anything).Return(client.AddEventReportResponse{
		EventReport: client.EventReport{
			Id:      "id",
			Uuid:    uuid,
			EventId: "event_id",
			Name:    "name",
		},
	}, nil)
	ctx := context.Background()
	titleMeta := plugindata.Map{
		"provider": plugindata.String("title"),
		"plugin":   plugindata.String("blackstork/builtin"),
	}

	diags := s.plugin.Publish(ctx, s.PublisherName(), &plugin.PublishParams{
		Config: plugintest.NewTestDecoder(s.T(), s.Publisher().Config).
			SetAttr("base_url", cty.StringVal("test")).
			SetAttr("api_key", cty.StringVal("test")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.Publisher().Args).
			SetAttr("event_id", cty.StringVal("event_id")).
			SetAttr("name", cty.StringVal("name")).
			SetAttr("distribution", cty.StringVal("0")).
			Decode(),
		Format: "md",
		DataContext: plugindata.Map{
			"document": plugindata.Map{
				"meta": plugindata.Map{
					"name": plugindata.String("test_document"),
				},
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
					},
				},
			},
		},
		DocumentName: "test_doc",
	})
	s.Require().Nil(diags)
}

func (s *MispPublishEventReportsTestSuite) TestError() {
	s.cli.On("AddEventReport", mock.Anything, mock.Anything).
		Return(client.AddEventReportResponse{}, errors.New("something went wrong"))
	ctx := context.Background()
	titleMeta := plugindata.Map{
		"provider": plugindata.String("title"),
		"plugin":   plugindata.String("blackstork/builtin"),
	}

	diags := s.plugin.Publish(ctx, s.PublisherName(), &plugin.PublishParams{
		Config: plugintest.NewTestDecoder(s.T(), s.Publisher().Config).
			SetAttr("base_url", cty.StringVal("test")).
			SetAttr("api_key", cty.StringVal("test")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.Publisher().Args).
			SetAttr("event_id", cty.StringVal("event_id")).
			SetAttr("name", cty.StringVal("name")).
			SetAttr("distribution", cty.StringVal("0")).
			Decode(),
		Format: "md",
		DataContext: plugindata.Map{
			"document": plugindata.Map{
				"meta": plugindata.Map{
					"name": plugindata.String("test_document"),
				},
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
					},
				},
			},
		},
		DocumentName: "test_doc",
	})
	diagtest.Asserts{{
		diagtest.IsError,
		diagtest.DetailContains("something went wrong"),
	}}.AssertMatch(s.T(), diags, nil)
}
