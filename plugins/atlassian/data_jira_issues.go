package atlassian

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/atlassian/client"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

func makeJiraIssuesDataSource(loader ClientLoadFn) *plugin.DataSource {
	return &plugin.DataSource{
		Doc:      "Retrieve issues from Jira.",
		DataFunc: searchJiraIssuesData(loader),
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "domain",
					Type:        cty.String,
					Constraints: constraint.RequiredMeaningful,
					Doc:         "Account Domain.",
				},
				{
					Name:        "account_email",
					Type:        cty.String,
					Secret:      true,
					Constraints: constraint.RequiredMeaningful,
					Doc:         "Account Email.",
				},
				{
					Name:        "api_token",
					Type:        cty.String,
					Secret:      true,
					Constraints: constraint.RequiredMeaningful,
					Doc:         "API Token.",
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "expand",
					Type: cty.String,
					Doc:  "Use expand to include additional information about issues in the response.",
					OneOf: constraint.OneOf{
						cty.StringVal("renderedFields"),
						cty.StringVal("names"),
						cty.StringVal("schema"),
						cty.StringVal("changelog"),
					},
					ExampleVal: cty.StringVal("names"),
				},
				{
					Name: "fields",
					Type: cty.List(cty.String),
					Doc:  "A list of fields to return for each issue.",
					ExampleVal: cty.ListVal([]cty.Value{
						cty.StringVal("*all"),
					}),
				},
				{
					Name:       "jql",
					Type:       cty.String,
					Doc:        "A JQL expression. For performance reasons, this field requires a bounded query. A bounded query is a query with a search restriction.",
					ExampleVal: cty.StringVal("order by key desc"),
				},
				{
					Name:         "properties",
					Type:         cty.List(cty.String),
					Doc:          "A list of up to 5 issue properties to include in the results.",
					MaxInclusive: cty.NumberIntVal(5),
					DefaultVal:   cty.ListValEmpty(cty.String),
				},
				{
					Name:         "size",
					Type:         cty.Number,
					Doc:          "Size limit to retrieve.",
					MinInclusive: cty.NumberIntVal(0),
					Constraints:  constraint.NonNull,
					DefaultVal:   cty.NumberIntVal(0),
				},
			},
		},
	}
}

func searchJiraIssuesData(loader ClientLoadFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		cli, err := parseConfig(params.Config, loader)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse configuration",
			}}
		}
		req, err := parseSearchIssuesReq(params.Args)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse arguments",
			}}
		}

		num, _ := params.Args.GetAttrVal("size").AsBigFloat().Int64()
		size := int(num)

		var issues plugindata.List
		for {
			res, err := cli.SearchIssues(ctx, req)
			if err != nil {
				return nil, handleClientError(err)
			}
			for _, v := range res.Issues {
				data, err := plugindata.ParseAny(v)
				if err != nil {
					return nil, diagnostics.Diag{{
						Severity: hcl.DiagError,
						Summary:  "Failed to parse data",
					}}
				}
				issues = append(issues, data)
				if size > 0 && len(issues) == size {
					break
				}
			}
			if (size > 0 && len(issues) == size) || res.NextPageToken == nil {
				break
			}
			req.NextPageToken = res.NextPageToken
		}
		return issues, nil
	}
}

func parseSearchIssuesReq(args *dataspec.Block) (*client.SearchIssuesReq, error) {
	req := &client.SearchIssuesReq{}
	if attr := args.GetAttrVal("expand"); !attr.IsNull() {
		req.Expand = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("jql"); !attr.IsNull() {
		req.JQL = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("fields"); !attr.IsNull() {
		for _, field := range attr.AsValueSlice() {
			if field.IsNull() {
				continue
			}
			req.Fields = append(req.Fields, field.AsString())
		}
	}
	for _, property := range args.GetAttrVal("properties").AsValueSlice() {
		if property.IsNull() {
			continue
		}
		req.Properties = append(req.Properties, property.AsString())
	}
	return req, nil
}
