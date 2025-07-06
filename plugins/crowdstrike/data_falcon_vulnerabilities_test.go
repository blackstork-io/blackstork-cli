package crowdstrike_test

import (
	"context"
	"errors"
	"testing"

	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client/spotlight_vulnerabilities"
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

type CrowdstrikeVulnerabilitiesTestSuite struct {
	suite.Suite
	plugin             *plugin.Schema
	cli                *mocks.Client
	vulnerabilitiesCli *mocks.SpotVulnerabilitiesClient
}

func TestCrowdstrikeVulnerabilitiesTestSuite(t *testing.T) {
	suite.Run(t, &CrowdstrikeVulnerabilitiesTestSuite{})
}

func (s *CrowdstrikeVulnerabilitiesTestSuite) SetupSuite() {
	s.plugin = crowdstrike.Plugin("1.2.3", func(cfg *falcon.ApiConfig) (client crowdstrike.Client, err error) {
		return s.cli, nil
	})
}

func (s *CrowdstrikeVulnerabilitiesTestSuite) SetupTest() {
	s.cli = &mocks.Client{}
	s.vulnerabilitiesCli = &mocks.SpotVulnerabilitiesClient{}
}

func (s *CrowdstrikeVulnerabilitiesTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *CrowdstrikeVulnerabilitiesTestSuite) TestSchema() {
	schema := s.plugin.DataSources[s.DatasourceName()]
	s.Require().NotNil(schema)
	s.NotNil(schema.Config)
	s.NotNil(schema.Args)
	s.NotNil(schema.DataFunc)
}

func (s *CrowdstrikeVulnerabilitiesTestSuite) DatasourceName() string {
	return "falcon_vulnerabilities"
}

func (s *CrowdstrikeVulnerabilitiesTestSuite) Datasource() *plugin.DataSource {
	return s.plugin.DataSources[s.DatasourceName()]
}

func (s *CrowdstrikeVulnerabilitiesTestSuite) String(val string) *string {
	return &val
}

func (s *CrowdstrikeVulnerabilitiesTestSuite) TestBasic() {
	s.cli.On("SpotlightVulnerabilities").Return(s.vulnerabilitiesCli)
	s.vulnerabilitiesCli.On("CombinedQueryVulnerabilities", mock.MatchedBy(func(params *spotlight_vulnerabilities.CombinedQueryVulnerabilitiesParams) bool {
		return params.Limit != nil && *params.Limit == 10
	})).Return(&spotlight_vulnerabilities.CombinedQueryVulnerabilitiesOK{
		Payload: &models.DomainSPAPICombinedVulnerabilitiesResponse{
			Resources: []*models.DomainBaseAPIVulnerabilityV2{
				{
					ID:     s.String("test_id_1"),
					Cid:    s.String("test_cid_1"),
					Status: s.String("test_status_1"),
				},
				{
					ID:     s.String("test_id_2"),
					Cid:    s.String("test_cid_2"),
					Status: s.String("test_status_2"),
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
		"cid":    plugindata.String("test_cid_1"),
		"id":     plugindata.String("test_id_1"),
		"status": plugindata.String("test_status_1"),
	})
	s.Subset(list[1], plugindata.Map{
		"cid":    plugindata.String("test_cid_2"),
		"id":     plugindata.String("test_id_2"),
		"status": plugindata.String("test_status_2"),
	})
}

func (s *CrowdstrikeVulnerabilitiesTestSuite) TestPayloadErrors() {
	s.cli.On("SpotlightVulnerabilities").Return(s.vulnerabilitiesCli)
	s.vulnerabilitiesCli.On("CombinedQueryVulnerabilities", mock.MatchedBy(func(params *spotlight_vulnerabilities.CombinedQueryVulnerabilitiesParams) bool {
		return params.Limit != nil && *params.Limit == 10
	})).Return(&spotlight_vulnerabilities.CombinedQueryVulnerabilitiesOK{
		Payload: &models.DomainSPAPICombinedVulnerabilitiesResponse{
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
		diagtest.SummaryContains("Failed to fetch Falcon Spotlight vulnerabilities"),
		diagtest.DetailContains("something went wrong"),
	}}.AssertMatch(s.T(), diags, nil)
}

func (s *CrowdstrikeVulnerabilitiesTestSuite) TestError() {
	s.cli.On("SpotlightVulnerabilities").Return(s.vulnerabilitiesCli)
	s.vulnerabilitiesCli.On("CombinedQueryVulnerabilities", mock.MatchedBy(func(params *spotlight_vulnerabilities.CombinedQueryVulnerabilitiesParams) bool {
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
		diagtest.SummaryContains("Failed to fetch Falcon Spotlight vulnerabilities"),
		diagtest.DetailContains("something went wrong"),
	}}.AssertMatch(s.T(), diags, nil)
}
