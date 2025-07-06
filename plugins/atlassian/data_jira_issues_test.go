package atlassian

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/atlassian/client"
	client_mocks "github.com/blackstork-io/fabric/mocks/internalpkg/atlassian/client"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
	"github.com/blackstork-io/fabric/internal/plugin/plugintest"
)

type JiraIssuesDataSourceTestSuite struct {
	suite.Suite

	plugin             *plugin.Schema
	ctx                context.Context
	cli                *client_mocks.Client
	storedApiURL       string
	storedAccountEmail string
	storedApiToken     string
}

func TestJiraIssuesDataSourceTestSuite(t *testing.T) {
	suite.Run(t, new(JiraIssuesDataSourceTestSuite))
}

func (s *JiraIssuesDataSourceTestSuite) SetupSuite() {
	s.plugin = Plugin("v0.0.0", func(apiURL, accountEmail, apiToken string) client.Client {
		s.storedApiURL = apiURL
		s.storedAccountEmail = accountEmail
		s.storedApiToken = apiToken
		return s.cli
	})
	s.ctx = context.Background()
}

func (s *JiraIssuesDataSourceTestSuite) SetupTest() {
	s.cli = &client_mocks.Client{}
}

func (s *JiraIssuesDataSourceTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *JiraIssuesDataSourceTestSuite) TestSchema() {
	s.Require().NotNil(s.plugin.DataSources["jira_issues"])
	s.NotNil(s.plugin.DataSources["jira_issues"].Config)
	s.NotNil(s.plugin.DataSources["jira_issues"].Args)
	s.NotNil(s.plugin.DataSources["jira_issues"].DataFunc)
}

func (s *JiraIssuesDataSourceTestSuite) TestLimit() {
	s.cli.On("SearchIssues", mock.Anything, &client.SearchIssuesReq{}).Return(&client.SearchIssuesRes{
		Issues: []any{
			map[string]any{
				"id": "1",
			},
		},
	}, nil)
	res, diags := s.plugin.RetrieveData(s.ctx, "jira_issues", &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["jira_issues"].Config).
			SetAttr("domain", cty.StringVal("test_domain")).
			SetAttr("account_email", cty.StringVal("test_account_email")).
			SetAttr("api_token", cty.StringVal("test_api_token")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["jira_issues"].Args).
			Decode(),
	})
	s.Equal("https://test_domain.atlassian.net", s.storedApiURL)
	s.Equal("test_account_email", s.storedAccountEmail)
	s.Equal("test_api_token", s.storedApiToken)
	s.Len(diags, 0)
	s.Equal(plugindata.List{
		plugindata.Map{
			"id": plugindata.String("1"),
		},
	}, res)
}

func (s *JiraIssuesDataSourceTestSuite) TestFull() {
	s.cli.On("SearchIssues", mock.Anything, &client.SearchIssuesReq{
		Expand:     client.String("names"),
		Fields:     []string{"*all"},
		JQL:        client.String("project = TEST"),
		Properties: []string{"example"},
	}).Return(&client.SearchIssuesRes{
		NextPageToken: client.String("page_2"),
		Issues: []any{
			map[string]any{
				"id": "1",
			},
		},
	}, nil)
	s.cli.On("SearchIssues", mock.Anything, &client.SearchIssuesReq{
		NextPageToken: client.String("page_2"),
		Expand:        client.String("names"),
		Fields:        []string{"*all"},
		JQL:           client.String("project = TEST"),
		Properties:    []string{"example"},
	}).Return(&client.SearchIssuesRes{
		Issues: []any{
			map[string]any{
				"id": "2",
			},
			map[string]any{
				"id": "3",
			},
		},
	}, nil)
	res, diags := s.plugin.RetrieveData(s.ctx, "jira_issues", &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["jira_issues"].Config).
			SetAttr("domain", cty.StringVal("test_domain")).
			SetAttr("account_email", cty.StringVal("test_account_email")).
			SetAttr("api_token", cty.StringVal("test_api_token")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["jira_issues"].Args).
			SetAttr("expand", cty.StringVal("names")).
			SetAttr("fields", cty.ListVal([]cty.Value{
				cty.StringVal("*all"),
			})).
			SetAttr("properties", cty.ListVal([]cty.Value{
				cty.StringVal("example"),
			})).
			SetAttr("jql", cty.StringVal("project = TEST")).
			SetAttr("size", cty.NumberIntVal(2)).
			Decode(),
	})
	s.Equal("https://test_domain.atlassian.net", s.storedApiURL)
	s.Equal("test_account_email", s.storedAccountEmail)
	s.Equal("test_api_token", s.storedApiToken)
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
