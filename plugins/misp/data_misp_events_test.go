package misp_test

import (
	"context"
	"errors"
	"testing"

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

type MispEventsDataTestSuite struct {
	suite.Suite
	plugin *plugin.Schema
	cli    *mocks.Client
}

func TestMispEventsDataTestSuite(t *testing.T) {
	suite.Run(t, &MispEventsDataTestSuite{})
}

func (s *MispEventsDataTestSuite) SetupSuite() {
	s.plugin = misp.Plugin("1.2.3", func(cfg *dataspec.Block) misp.Client {
		return s.cli
	})
}

func (s *MispEventsDataTestSuite) DatasourceName() string {
	return "misp_events"
}

func (s *MispEventsDataTestSuite) Datasource() *plugin.DataSource {
	return s.plugin.DataSources[s.DatasourceName()]
}

func (s *MispEventsDataTestSuite) SetupTest() {
	s.cli = &mocks.Client{}
}

func (s *MispEventsDataTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *MispEventsDataTestSuite) TestSchema() {
	schema := s.Datasource()
	s.Require().NotNil(schema)
	s.NotNil(schema.Config)
	s.NotNil(schema.Args)
	s.NotNil(schema.DataFunc)
}

func (s *MispEventsDataTestSuite) String(val string) *string {
	return &val
}

func (s *MispEventsDataTestSuite) TestBasic() {
	s.cli.On("RestSearchEvents", mock.Anything, mock.Anything).Return(client.RestSearchEventsResponse{
		Response: []client.EventResponse{
			{
				Event: client.Event{
					ID:   "1",
					Date: "2021-01-01",
				},
			},
			{
				Event: client.Event{
					ID:   "2",
					Date: "2022-01-01",
				},
			},
		},
	}, nil)
	ctx := context.Background()
	data, diags := s.plugin.RetrieveData(ctx, s.DatasourceName(), &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.Datasource().Config).
			SetAttr("base_url", cty.StringVal("test")).
			SetAttr("api_key", cty.StringVal("test")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.Datasource().Args).
			SetAttr("limit", cty.NumberIntVal(10)).
			SetAttr("value", cty.StringVal("test_filter")).
			Decode(),
	})
	s.Require().Nil(diags)
	dataMap := data.AsPluginData().(plugindata.Map)
	list := dataMap["response"].AsPluginData().(plugindata.List)
	s.Len(list, 2)
	event1 := list[0].AsPluginData().(plugindata.Map)["Event"].AsPluginData().(plugindata.Map)
	s.Subset(event1, plugindata.Map{
		"id":   plugindata.String("1"),
		"date": plugindata.String("2021-01-01"),
	})
	event2 := list[1].AsPluginData().(plugindata.Map)["Event"].AsPluginData().(plugindata.Map)
	s.Subset(event2, plugindata.Map{
		"id":   plugindata.String("2"),
		"date": plugindata.String("2022-01-01"),
	})
}

func (s *MispEventsDataTestSuite) TestApiError() {
	s.cli.On("RestSearchEvents", mock.Anything, mock.Anything).Return(client.RestSearchEventsResponse{}, errors.New("something went wrong"))
	ctx := context.Background()
	_, diags := s.plugin.RetrieveData(ctx, s.DatasourceName(), &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.Datasource().Config).
			SetAttr("base_url", cty.StringVal("test")).
			SetAttr("api_key", cty.StringVal("test")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.Datasource().Args).
			SetAttr("limit", cty.NumberIntVal(10)).
			SetAttr("value", cty.StringVal("test_filter")).
			Decode(),
	})
	diagtest.Asserts{{
		diagtest.IsError,
		diagtest.SummaryContains("Failed to fetch events"),
		diagtest.DetailContains("something went wrong"),
	}}.AssertMatch(s.T(), diags, nil)
}
