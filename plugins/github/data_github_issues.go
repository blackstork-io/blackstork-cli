package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	gh "github.com/google/go-github/v58/github"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/utils"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

func makeGithubIssuesDataSchema(loader ClientLoaderFn) *plugin.DataSource {
	return &plugin.DataSource{
		DataFunc: fetchGithubIssuesData(loader),
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "github_token",
					Type:        cty.String,
					Constraints: constraint.RequiredMeaningful,
					Secret:      true,
					Doc:         "The GitHub token to use for authentication",
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "repository",
					Type:        cty.String,
					Constraints: constraint.RequiredMeaningful,
					ExampleVal:  cty.StringVal("blackstork-io/fabric"),
					Doc:         "The repository to list issues from, in the format of owner/name",
				},
				{
					Name:        "milestone",
					Type:        cty.String,
					Constraints: constraint.TrimSpace | constraint.NonNull,
					Doc: utils.Dedent(`
						Filter issues by milestone. Possible values are:
						* a milestone number
						* "none" for issues with no milestone
						* "*" for issues with any milestone
						* "" (empty string) performs no filtering
					`),
					DefaultVal: cty.StringVal(""),
				},
				{
					Name:        "state",
					Type:        cty.String,
					Constraints: constraint.Meaningful,
					Doc:         `Filter issues based on their state`,
					OneOf: []cty.Value{
						cty.StringVal("open"),
						cty.StringVal("closed"),
						cty.StringVal("all"),
					},
					DefaultVal: cty.StringVal("open"),
				},
				{
					Name:        "assignee",
					Type:        cty.String,
					Constraints: constraint.TrimSpace | constraint.NonNull,
					Doc: utils.Dedent(`
						Filter issues based on their assignee. Possible values are:
						* a user name
						* "none" for issues that are not assigned
						* "*" for issues with any assigned user
						* "" (empty string) performs no filtering.
					`),
					DefaultVal: cty.StringVal(""),
				},
				{
					Name:        "creator",
					Type:        cty.String,
					Constraints: constraint.TrimSpace | constraint.NonNull,
					Doc: utils.Dedent(`
						Filter issues based on their creator. Possible values are:
						* a user name
						* "" (empty string) performs no filtering.
					`),
					DefaultVal: cty.StringVal(""),
				},
				{
					Name:        "mentioned",
					Type:        cty.String,
					Constraints: constraint.TrimSpace | constraint.NonNull,
					Doc: utils.Dedent(`
						Filter issues to once where this username is mentioned. Possible values are:
						* a user name
						* "" (empty string) performs no filtering.
					`),
					DefaultVal: cty.StringVal(""),
				},
				{
					Name:       "labels",
					Type:       cty.List(cty.String),
					Doc:        `Filter issues based on their labels.`,
					DefaultVal: cty.NullVal(cty.List(cty.String)),
				},
				{
					Name:        "sort",
					Type:        cty.String,
					Constraints: constraint.Meaningful,
					Doc:         `Specifies how to sort issues.`,
					OneOf: []cty.Value{
						cty.StringVal("created"),
						cty.StringVal("updated"),
						cty.StringVal("comments"),
					},
					DefaultVal: cty.StringVal("created"),
				},
				{
					Name:        "direction",
					Type:        cty.String,
					Constraints: constraint.Meaningful,
					Doc:         `Specifies the direction in which to sort issues.`,
					OneOf: []cty.Value{
						cty.StringVal("asc"),
						cty.StringVal("desc"),
					},
					DefaultVal: cty.StringVal("desc"),
				},
				{
					Name:        "since",
					Type:        cty.String,
					Constraints: constraint.Meaningful,
					Doc: utils.Dedent(`
						Only show results that were last updated after the given time.
						This is a timestamp in ISO 8601 format: YYYY-MM-DDTHH:MM:SSZ.
					`),
				},
				{
					Name:         "limit",
					Type:         cty.Number,
					Constraints:  constraint.Integer,
					MinInclusive: cty.NumberIntVal(-1),
					Doc:          "Limit the number of issues to return. -1 means no limit.",
					DefaultVal:   cty.NumberIntVal(-1),
				},
			},
		},
	}
}

func fetchGithubIssuesData(loader ClientLoaderFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		tkn := params.Config.GetAttrVal("github_token").AsString()
		opts, diags := parseIssuesArgs(params.Args)
		if diags.HasErrors() {
			return nil, diags
		}
		client := loader(tkn)
		// iterate over pages until we get all issues or reach the limit if specified
		var issues plugindata.List
		for page := minPage; ; page++ {
			opts.opts.Page = page
			opts.opts.PerPage = pageSize
			issuesPage, resp, err := client.Issues().ListByRepo(context.Background(), opts.owner, opts.name, opts.opts)
			if err != nil {
				return nil, diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Failed to list issues",
					Detail:   err.Error(),
				}}
			}
			for _, issue := range issuesPage {
				encoded, err := encodeGithubIssue(issue)
				if err != nil {
					return nil, diagnostics.Diag{{
						Severity: hcl.DiagError,
						Summary:  "Failed to encode issue",
						Detail:   err.Error(),
					}}
				}
				issues = append(issues, encoded)
			}
			if resp.NextPage == 0 || (opts.limit > 0 && int64(len(issues)) >= opts.limit) {
				break
			}
		}
		// if limit is specified, truncate the issues slice
		if opts.limit > 0 && int64(len(issues)) > opts.limit {
			issues = issues[:opts.limit]
		}
		return issues, nil
	}
}

type parsedIssuesArgs struct {
	owner string
	name  string
	limit int64
	opts  *gh.IssueListByRepoOptions
}

func parseIssuesArgs(args *dataspec.Block) (*parsedIssuesArgs, diagnostics.Diag) {
	parsed := &parsedIssuesArgs{
		opts: &gh.IssueListByRepoOptions{
			Milestone: args.GetAttrVal("milestone").AsString(),
			State:     args.GetAttrVal("state").AsString(),
			Assignee:  args.GetAttrVal("assignee").AsString(),
			Creator:   args.GetAttrVal("creator").AsString(),
			Mentioned: args.GetAttrVal("mentioned").AsString(),
			Sort:      args.GetAttrVal("sort").AsString(),
			Direction: args.GetAttrVal("direction").AsString(),
		},
	}
	parsed.limit, _ = args.GetAttrVal("limit").AsBigFloat().Int64()

	repo := args.Attrs["repository"]
	repositoryParts := strings.Split(repo.Value.AsString(), "/")
	if len(repositoryParts) != 2 {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Invalid repository format",
			Detail:   "Repository must be in the format of owner/name",
			Subject:  &repo.ValueRange,
		}}
	}
	parsed.owner = repositoryParts[0]
	parsed.name = repositoryParts[1]
	if labels := args.GetAttrVal("labels"); !labels.IsNull() {
		parsed.opts.Labels = make([]string, labels.LengthInt())
		for i, label := range labels.AsValueSlice() {
			parsed.opts.Labels[i] = label.AsString()
		}
	}
	since := args.Attrs["since"]
	if since != nil {
		ts, err := time.Parse(time.RFC3339, since.Value.AsString())
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Invalid timestamp format",
				Detail: fmt.Sprintf(
					"Timestamp must be in ISO 8601 format: YYYY-MM-DDTHH:MM:SSZ, but %s",
					err.Error(),
				),
				Subject: &since.ValueRange,
			}}
		}
		parsed.opts.Since = ts
	}

	return parsed, nil
}

func encodeGithubIssue(issue *gh.Issue) (plugindata.Data, error) {
	raw, err := json.Marshal(issue)
	if err != nil {
		return nil, fmt.Errorf("failed to encode issue: %w", err)
	}
	return plugindata.UnmarshalJSON(raw)
}
