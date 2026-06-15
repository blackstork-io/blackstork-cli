// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package github

import (
	"context"

	gh "github.com/google/go-github/v84/github"

	"github.com/blackstork-io/blackstork-cli/plugin"
)

var DefaultClientLoader = func(token string) Client {
	return &ClientAdapter{gh.NewClient(nil).WithAuthToken(token)}
}

const (
	minPage  = 1
	pageSize = 30
)

type ClientLoaderFn func(token string) Client

type Client interface {
	Issues() IssuesClient
	Gists() GistClient
}

type ClientAdapter struct {
	gh *gh.Client
}

func (c *ClientAdapter) Issues() IssuesClient {
	return c.gh.Issues
}

func (c *ClientAdapter) Gists() GistClient {
	return c.gh.Gists
}

type IssuesClient interface {
	ListByRepo(ctx context.Context, owner, repo string, opts *gh.IssueListByRepoOptions) ([]*gh.Issue, *gh.Response, error)
}

type GistClient interface {
	Create(ctx context.Context, gist *gh.Gist) (*gh.Gist, *gh.Response, error)
	Get(ctx context.Context, id string) (*gh.Gist, *gh.Response, error)
	Edit(ctx context.Context, id string, gist *gh.Gist) (*gh.Gist, *gh.Response, error)
}

func Plugin(version string, clientLoader ClientLoaderFn) *plugin.Schema {
	if clientLoader == nil {
		clientLoader = DefaultClientLoader
	}
	return &plugin.Schema{
		Name:    "blackstork/github",
		Version: version,
		DataSources: plugin.DataSources{
			"github_issues": makeGithubIssuesDataSchema(clientLoader),
		},
		Publishers: plugin.Publishers{
			"github_gist": makeGithubGistPublisher(clientLoader),
		},
	}
}
