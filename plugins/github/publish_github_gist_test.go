package github_test

import (
	"context"
	"errors"
	"testing"

	gh "github.com/google/go-github/v58/github"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/github"
	github_mocks "github.com/blackstork-io/fabric/mocks/internalpkg/github"
	"github.com/blackstork-io/fabric/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/plugindata"
	"github.com/blackstork-io/fabric/plugin/plugintest"
)

type GithubPublishGistTestSuite struct {
	suite.Suite
	plugin  *plugin.Schema
	cli     *github_mocks.Client
	gistCli *github_mocks.GistClient
}

func TestGithubublishGistSuite(t *testing.T) {
	suite.Run(t, &GithubPublishGistTestSuite{})
}

func (s *GithubPublishGistTestSuite) SetupSuite() {
	s.plugin = github.Plugin("1.2.3", func(token string) github.Client {
		return s.cli
	})
}

func (s *GithubPublishGistTestSuite) SetupTest() {
	s.cli = &github_mocks.Client{}
	s.gistCli = &github_mocks.GistClient{}
}

func (s *GithubPublishGistTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *GithubPublishGistTestSuite) TestSchema() {
	schema := s.plugin.Publishers["github_gist"]
	s.Require().NotNil(schema)
	s.NotNil(schema.Config)
	s.NotNil(schema.Args)
	s.NotNil(schema.PublishFunc)
}

func (s *GithubPublishGistTestSuite) TestBasic() {
	s.cli.On("Gists").Return(s.gistCli)
	s.gistCli.On("Create", mock.Anything, &gh.Gist{
		Description: gh.String("test description"),
		Public:      gh.Bool(false),
		Files: map[gh.GistFilename]gh.GistFile{
			"filename.md": {
				Content:  gh.String("# Header 1\n\nLorem ipsum dolor sit amet, consectetur adipiscing elit."),
				Filename: gh.String("filename.md"),
			},
		},
	}).Return(&gh.Gist{HTMLURL: gh.String("http://gist.github.com/mock")}, &gh.Response{}, nil)
	ctx := context.Background()
	titleMeta := plugindata.Map{
		"provider": plugindata.String("title"),
		"plugin":   plugindata.String("blackstork/builtin"),
	}

	diags := s.plugin.Publish(ctx, "github_gist", &plugin.PublishParams{
		Config: plugintest.NewTestDecoder(s.T(), s.plugin.Publishers["github_gist"].Config).
			SetAttr("github_token", cty.StringVal("testtoken")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.plugin.Publishers["github_gist"].Args).
			SetAttr("description", cty.StringVal("test description")).
			SetAttr("filename", cty.StringVal("filename.md")).
			Decode(),
		Format: plugin.OutputFormatMD,
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

func (s *GithubPublishGistTestSuite) TestFilenameOptional() {
	s.cli.On("Gists").Return(s.gistCli)
	s.gistCli.On("Create", mock.Anything, &gh.Gist{
		Description: gh.String("test description"),
		Public:      gh.Bool(false),
		Files: map[gh.GistFilename]gh.GistFile{
			"test_doc.md": {
				Content:  gh.String("# Header 1\n\nLorem ipsum dolor sit amet, consectetur adipiscing elit."),
				Filename: gh.String("test_doc.md"),
			},
		},
	}).Return(&gh.Gist{HTMLURL: gh.String("http://gist.github.com/mock")}, &gh.Response{}, nil)
	ctx := context.Background()
	titleMeta := plugindata.Map{
		"provider": plugindata.String("title"),
		"plugin":   plugindata.String("blackstork/builtin"),
	}
	diags := s.plugin.Publish(ctx, "github_gist", &plugin.PublishParams{
		Config: plugintest.NewTestDecoder(s.T(), s.plugin.Publishers["github_gist"].Config).
			SetAttr("github_token", cty.StringVal("testtoken")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.plugin.Publishers["github_gist"].Args).
			SetAttr("description", cty.StringVal("test description")).
			SetAttr("filename", cty.StringVal("")).
			Decode(),

		Format: plugin.OutputFormatMD,
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

func (s *GithubPublishGistTestSuite) TestResetOldFiles() {
	s.cli.On("Gists").Return(s.gistCli)
	s.gistCli.On("Get", mock.Anything, "gistid").Return(&gh.Gist{
		Files: map[gh.GistFilename]gh.GistFile{
			"oldfile.md": {
				Filename: gh.String("oldfile.md"),
				Content:  gh.String("old content"),
			},
		},
	}, &gh.Response{}, nil)
	s.gistCli.On("Edit", mock.Anything, "gistid", &gh.Gist{
		Description: gh.String("test description"),
		Public:      gh.Bool(false),
		Files: map[gh.GistFilename]gh.GistFile{
			"oldfile.md": {},
			"test_doc.md": {
				Content:  gh.String("# Header 1\n\nLorem ipsum dolor sit amet, consectetur adipiscing elit."),
				Filename: gh.String("test_doc.md"),
			},
		},
	}).Return(&gh.Gist{HTMLURL: gh.String("http://gist.github.com/mock")}, &gh.Response{}, nil)
	ctx := context.Background()
	titleMeta := plugindata.Map{
		"provider": plugindata.String("title"),
		"plugin":   plugindata.String("blackstork/builtin"),
	}
	diags := s.plugin.Publish(ctx, "github_gist", &plugin.PublishParams{
		Config: plugintest.NewTestDecoder(s.T(), s.plugin.Publishers["github_gist"].Config).
			SetAttr("github_token", cty.StringVal("testtoken")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.plugin.Publishers["github_gist"].Args).
			SetAttr("description", cty.StringVal("test description")).
			SetAttr("gist_id", cty.StringVal("gistid")).
			Decode(),
		Format: plugin.OutputFormatMD,
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

func (s *GithubPublishGistTestSuite) TestError() {
	s.cli.On("Gists").Return(s.gistCli)
	s.gistCli.On("Create", mock.Anything, mock.Anything).Return(nil, nil, errors.New("github api failed"))
	ctx := context.Background()
	titleMeta := plugindata.Map{
		"provider": plugindata.String("title"),
		"plugin":   plugindata.String("blackstork/builtin"),
	}

	diags := s.plugin.Publish(ctx, "github_gist", &plugin.PublishParams{
		Config: plugintest.NewTestDecoder(s.T(), s.plugin.Publishers["github_gist"].Config).
			SetAttr("github_token", cty.StringVal("testtoken")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.plugin.Publishers["github_gist"].Args).
			SetAttr("description", cty.StringVal("test description")).
			SetAttr("filename", cty.StringVal("filename.md")).
			Decode(),
		Format: plugin.OutputFormatMD,
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
		diagtest.DetailContains("github api failed"),
	}}.AssertMatch(s.T(), diags, nil)
}
