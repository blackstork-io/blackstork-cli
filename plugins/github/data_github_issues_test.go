package github_test

import (
	"context"
	"testing"
	"time"

	gh "github.com/google/go-github/v58/github"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/github"
	github_mocks "github.com/blackstork-io/fabric/mocks/internalpkg/github"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/plugindata"
	"github.com/blackstork-io/fabric/plugin/plugintest"
)

type GithubIssuesDataTestSuite struct {
	suite.Suite
	plugin    *plugin.Schema
	cli       *github_mocks.Client
	issuesCli *github_mocks.IssuesClient
}

func TestGithubDataSuite(t *testing.T) {
	suite.Run(t, &GithubIssuesDataTestSuite{})
}

func (s *GithubIssuesDataTestSuite) SetupSuite() {
	s.plugin = github.Plugin("1.2.3", func(token string) github.Client {
		return s.cli
	})
}

func (s *GithubIssuesDataTestSuite) SetupTest() {
	s.cli = &github_mocks.Client{}
	s.issuesCli = &github_mocks.IssuesClient{}
}

func (s *GithubIssuesDataTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *GithubIssuesDataTestSuite) TestSchema() {
	schema := s.plugin.DataSources["github_issues"]
	s.Require().NotNil(schema)
	s.NotNil(schema.Config)
	s.NotNil(schema.Args)
	s.NotNil(schema.DataFunc)
}

func int64ptr(i int64) *int64 { return &i }

func (s *GithubIssuesDataTestSuite) TestBasic() {
	s.cli.On("Issues").Return(s.issuesCli)
	s.issuesCli.On("ListByRepo", mock.Anything, "testorg", "testrepo", &gh.IssueListByRepoOptions{
		Milestone: "", State: "open", Assignee: "", Creator: "", Mentioned: "", Labels: []string(nil), Sort: "created", Direction: "desc", Since: time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), ListOptions: gh.ListOptions{Page: 1, PerPage: 30},
	}).
		Return([]*gh.Issue{
			{
				ID: int64ptr(123),
			},
			{
				ID: int64ptr(124),
			},
		}, &gh.Response{}, nil)

	ctx := context.Background()
	data, diags := s.plugin.RetrieveData(ctx, "github_issues", &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["github_issues"].Config).
			SetHeaders("config").
			SetAttr("github_token", cty.StringVal("testtoken")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["github_issues"].Args).
			SetHeaders("data", "github_issues", `"test"`).
			SetAttr("repository", cty.StringVal("testorg/testrepo")).
			Decode(),
	})
	s.Require().Nil(diags)
	s.Equal(plugindata.List{
		plugindata.Map{
			"id": plugindata.Number(123),
		},
		plugindata.Map{
			"id": plugindata.Number(124),
		},
	}, data)
}

func (s *GithubIssuesDataTestSuite) TestAdvanced() {
	since, err := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
	s.Require().NoError(err)
	s.cli.On("Issues").Return(s.issuesCli)
	s.issuesCli.On("ListByRepo", mock.Anything, "testorg", "testrepo", &gh.IssueListByRepoOptions{
		Milestone: "testmilestone",
		State:     "open",
		Assignee:  "testassignee",
		Creator:   "testcreator",
		Labels: []string{
			"testlabel1",
			"testlabel2",
		},
		Sort:      "created",
		Direction: "asc",
		Mentioned: "testmentioned",
		Since:     since,
		ListOptions: gh.ListOptions{
			PerPage: 30,
			Page:    1,
		},
	}).
		Return([]*gh.Issue{
			{
				ID: int64ptr(123),
			},
			{
				ID: int64ptr(124),
			},
		}, &gh.Response{}, nil)

	ctx := context.Background()
	data, diags := s.plugin.RetrieveData(ctx, "github_issues", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"cfg"}, map[string]cty.Value{
			"github_token": cty.StringVal("testtoken"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"repository": cty.StringVal("testorg/testrepo"),
			"limit":      cty.NumberIntVal(2),
			"milestone":  cty.StringVal("testmilestone"),
			"state":      cty.StringVal("open"),
			"assignee":   cty.StringVal("testassignee"),
			"creator":    cty.StringVal("testcreator"),
			"mentioned":  cty.StringVal("testmentioned"),
			"labels": cty.ListVal([]cty.Value{
				cty.StringVal("testlabel1"),
				cty.StringVal("testlabel2"),
			}),
			"sort":      cty.StringVal("created"),
			"direction": cty.StringVal("asc"),
			"since":     cty.StringVal("2021-01-01T00:00:00Z"),
		}),
	})
	s.Require().Nil(diags)
	s.Equal(plugindata.List{
		plugindata.Map{
			"id": plugindata.Number(123),
		},
		plugindata.Map{
			"id": plugindata.Number(124),
		},
	}, data)
}
