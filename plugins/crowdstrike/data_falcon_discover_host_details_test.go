package crowdstrike_test

import (
	"context"
	"errors"
	"testing"

	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client/discover"
	"github.com/crowdstrike/gofalcon/falcon/models"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/crowdstrike"
	mocks "github.com/blackstork-io/fabric/mocks/internalpkg/crowdstrike"
	"github.com/blackstork-io/fabric/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
	"github.com/blackstork-io/fabric/internal/plugin/plugintest"
)

type CrowdstrikeDiscoverHostDetailsTestSuite struct {
	suite.Suite
	plugin      *plugin.Schema
	cli         *mocks.Client
	discoverCli *mocks.DiscoverClient
}

func TestCrowdstrikeDiscoverHostDetailsSuite(t *testing.T) {
	suite.Run(t, &CrowdstrikeDiscoverHostDetailsTestSuite{})
}

func (s *CrowdstrikeDiscoverHostDetailsTestSuite) SetupSuite() {
	s.plugin = crowdstrike.Plugin("1.2.3", func(cfg *falcon.ApiConfig) (client crowdstrike.Client, err error) {
		return s.cli, nil
	})
}

func (s *CrowdstrikeDiscoverHostDetailsTestSuite) SetupTest() {
	s.cli = &mocks.Client{}
	s.discoverCli = &mocks.DiscoverClient{}
}

func (s *CrowdstrikeDiscoverHostDetailsTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *CrowdstrikeDiscoverHostDetailsTestSuite) DatasourceName() string {
	return "falcon_discover_host_details"
}

func (s *CrowdstrikeDiscoverHostDetailsTestSuite) Datasource() *plugin.DataSource {
	return s.plugin.DataSources[s.DatasourceName()]
}

func (s *CrowdstrikeDiscoverHostDetailsTestSuite) TestSchema() {
	schema := s.plugin.DataSources["falcon_cspm_ioms"]
	s.Require().NotNil(schema)
	s.NotNil(schema.Config)
	s.NotNil(schema.Args)
	s.NotNil(schema.DataFunc)
}

func (s *CrowdstrikeDiscoverHostDetailsTestSuite) String(val string) *string {
	return &val
}

func (s *CrowdstrikeDiscoverHostDetailsTestSuite) TestBasic() {
	s.cli.On("Discover").Return(s.discoverCli)
	s.discoverCli.On("QueryHosts", mock.MatchedBy(func(params *discover.QueryHostsParams) bool {
		return params.Limit != nil && *params.Limit == 10
	})).Return(&discover.QueryHostsOK{
		Payload: &models.MsaspecQueryResponse{
			Resources: []string{"test_host_1", "test_host_2"},
		},
	}, nil)
	s.discoverCli.On("GetHosts", mock.MatchedBy(func(params *discover.GetHostsParams) bool {
		return params.Ids[0] == "test_host_1" && params.Ids[1] == "test_host_2"
	})).Return(&discover.GetHostsOK{
		Payload: &models.DomainDiscoverAPIHostEntitiesResponse{
			Resources: []*models.DomainDiscoverAPIHost{
				{
					Cid:  s.String("test_cid_1"),
					City: "Dublin",
				},
				{
					Cid:  s.String("test_cid_2"),
					City: "London",
				},
			},
		},
	}, nil)
	ctx := context.Background()
	data, diags := s.plugin.RetrieveData(ctx, s.DatasourceName(), &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.Datasource().Config).
			SetAttr("client_id", cty.StringVal("test")).
			SetAttr("client_secret", cty.StringVal("test")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.Datasource().Args).
			SetAttr("limit", cty.NumberIntVal(10)).
			Decode(),
	})
	s.Require().Nil(diags)
	list := data.AsPluginData().(plugindata.List)
	s.Len(list, 2)
	s.Subset(list[0], plugindata.Map{
		"cid":  plugindata.String("test_cid_1"),
		"city": plugindata.String("Dublin"),
	})
	s.Subset(list[1], plugindata.Map{
		"cid":  plugindata.String("test_cid_2"),
		"city": plugindata.String("London"),
	})
}

func (s *CrowdstrikeDiscoverHostDetailsTestSuite) TestPayloadErrors() {
	s.cli.On("Discover").Return(s.discoverCli)
	s.discoverCli.On("QueryHosts", mock.MatchedBy(func(params *discover.QueryHostsParams) bool {
		return params.Limit != nil && *params.Limit == 10
	})).Return(&discover.QueryHostsOK{
		Payload: &models.MsaspecQueryResponse{
			Errors: []*models.MsaAPIError{{
				Message: s.String("something went wrong"),
			}},
		},
	}, nil)
	ctx := context.Background()
	_, diags := s.plugin.RetrieveData(ctx, s.DatasourceName(), &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.Datasource().Config).
			SetAttr("client_id", cty.StringVal("test")).
			SetAttr("client_secret", cty.StringVal("test")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.Datasource().Args).
			SetAttr("limit", cty.NumberIntVal(10)).
			Decode(),
	})
	diagtest.Asserts{{
		diagtest.IsError,
		diagtest.SummaryContains("Failed to query Falcon Discover Hosts"),
		diagtest.DetailContains("something went wrong"),
	}}.AssertMatch(s.T(), diags, nil)
}

func (s *CrowdstrikeDiscoverHostDetailsTestSuite) TestError() {
	s.cli.On("Discover").Return(s.discoverCli)
	s.discoverCli.On("QueryHosts", mock.MatchedBy(func(params *discover.QueryHostsParams) bool {
		return params.Limit != nil && *params.Limit == 10
	})).Return(nil, errors.New("something went wrong"))
	ctx := context.Background()
	_, diags := s.plugin.RetrieveData(ctx, s.DatasourceName(), &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.Datasource().Config).
			SetAttr("client_id", cty.StringVal("test")).
			SetAttr("client_secret", cty.StringVal("test")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.Datasource().Args).
			SetAttr("limit", cty.NumberIntVal(10)).
			Decode(),
	})
	diagtest.Asserts{{
		diagtest.IsError,
		diagtest.SummaryContains("Failed to query Falcon Discover Hosts"),
		diagtest.DetailContains("something went wrong"),
	}}.AssertMatch(s.T(), diags, nil)
}
