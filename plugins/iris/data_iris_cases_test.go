package iris

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/iris/client"
	client_mocks "github.com/blackstork-io/fabric/mocks/internalpkg/iris/client"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
	"github.com/blackstork-io/fabric/internal/plugin/plugintest"
)

type CasesDataSourceTestSuite struct {
	suite.Suite

	plugin         *plugin.Schema
	ctx            context.Context
	cli            *client_mocks.Client
	storedApiURL   string
	storedApiKey   string
	storedInsecure bool
}

func TestCasesDataSourceTestSuite(t *testing.T) {
	suite.Run(t, new(CasesDataSourceTestSuite))
}

func (s *CasesDataSourceTestSuite) SetupSuite() {
	s.plugin = Plugin("v0.0.0", func(apiURL, apiKey string, insecure bool) client.Client {
		s.storedApiKey = apiKey
		s.storedApiURL = apiURL
		s.storedInsecure = insecure
		return s.cli
	})
	s.ctx = context.Background()
}

func (s *CasesDataSourceTestSuite) SetupTest() {
	s.cli = &client_mocks.Client{}
}

func (s *CasesDataSourceTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *CasesDataSourceTestSuite) TestSchema() {
	s.Require().NotNil(s.plugin.DataSources["iris_cases"])
	s.NotNil(s.plugin.DataSources["iris_cases"].Config)
	s.NotNil(s.plugin.DataSources["iris_cases"].Args)
	s.NotNil(s.plugin.DataSources["iris_cases"].DataFunc)
}

func (s *CasesDataSourceTestSuite) TestLimit() {
	s.cli.On("ListCases", mock.Anything, &client.ListCasesReq{
		Page: 1,
		Sort: client.String("desc"),
	}).Return(&client.ListCasesRes{
		Status: "success",
		Data: &client.CasesData{
			CurrentPage: 1,
			LastPage:    1,
			Total:       1,
			Cases: []any{
				map[string]any{
					"id": "1",
				},
			},
		},
	}, nil)
	res, diags := s.plugin.RetrieveData(s.ctx, "iris_cases", &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["iris_cases"].Config).
			SetAttr("api_url", cty.StringVal("test-url")).
			SetAttr("api_key", cty.StringVal("test-key")).
			SetAttr("insecure", cty.BoolVal(true)).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["iris_cases"].Args).
			SetAttr("size", cty.NumberIntVal(10)).
			Decode(),
	})
	s.Equal("test-url", s.storedApiURL)
	s.Equal("test-key", s.storedApiKey)
	s.Equal(true, s.storedInsecure)
	s.Len(diags, 0)
	s.Equal(plugindata.List{
		plugindata.Map{
			"id": plugindata.String("1"),
		},
	}, res)
}

func (s *CasesDataSourceTestSuite) TestFull() {
	s.cli.On("ListCases", mock.Anything, &client.ListCasesReq{
		Page:           1,
		CaseIDs:        client.IntList{1, 2},
		CaseCustomerID: client.Int(1),
		CaseOwnerID:    client.Int(2),
		CaseSeverityID: client.Int(4),
		CaseStateID:    client.Int(3),
		CaseSocID:      client.String("test-soc"),
		Sort:           client.String("asc"),
		StartOpenDate:  client.String("test-start-open-date"),
		EndOpenDate:    client.String("test-end-open-date"),
	}).Return(&client.ListCasesRes{
		Status: "success",
		Data: &client.CasesData{
			CurrentPage: 1,
			LastPage:    2,
			Total:       3,
			NextPage:    client.Int(2),
			Cases: []any{
				map[string]any{
					"id": "1",
				},
			},
		},
	}, nil)
	s.cli.On("ListCases", mock.Anything, &client.ListCasesReq{
		Page:           2,
		CaseIDs:        client.IntList{1, 2},
		CaseCustomerID: client.Int(1),
		CaseOwnerID:    client.Int(2),
		CaseSeverityID: client.Int(4),
		CaseStateID:    client.Int(3),
		CaseSocID:      client.String("test-soc"),
		Sort:           client.String("asc"),
		StartOpenDate:  client.String("test-start-open-date"),
		EndOpenDate:    client.String("test-end-open-date"),
	}).Return(&client.ListCasesRes{
		Status: "success",
		Data: &client.CasesData{
			CurrentPage: 2,
			LastPage:    2,
			Total:       3,
			Cases: []any{
				map[string]any{
					"id": "2",
				},
				map[string]any{
					"id": "3",
				},
			},
		},
	}, nil)
	res, diags := s.plugin.RetrieveData(s.ctx, "iris_cases", &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["iris_cases"].Config).
			SetAttr("api_url", cty.StringVal("test-url")).
			SetAttr("api_key", cty.StringVal("test-key")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["iris_cases"].Args).
			SetAttr("case_ids", cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
			})).
			SetAttr("customer_id", cty.NumberIntVal(1)).
			SetAttr("owner_id", cty.NumberIntVal(2)).
			SetAttr("severity_id", cty.NumberIntVal(4)).
			SetAttr("state_id", cty.NumberIntVal(3)).
			SetAttr("soc_id", cty.StringVal("test-soc")).
			SetAttr("start_open_date", cty.StringVal("test-start-open-date")).
			SetAttr("end_open_date", cty.StringVal("test-end-open-date")).
			SetAttr("sort", cty.StringVal("asc")).
			SetAttr("size", cty.NumberIntVal(2)).
			Decode(),
	})
	s.Equal("test-url", s.storedApiURL)
	s.Equal("test-key", s.storedApiKey)
	s.Equal(false, s.storedInsecure)
	s.Len(diags, 0)
	s.Equal(plugindata.List{
		plugindata.Map{
			"id": plugindata.String("1"),
		},
		plugindata.Map{
			"id": plugindata.String("2"),
		},
	}, res)
}
