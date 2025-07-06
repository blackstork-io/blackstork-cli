package microsoft

import (
	"context"
	"fmt"

	"github.com/google/go-querystring/query"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

type GetIncidentsReq struct {
	SubscriptionID    string  `url:"-"`
	ResourceGroupName string  `url:"-"`
	WorkspaceName     string  `url:"-"`
	Size              int     `url:"-"`
	Filter            *string `url:"$filter,omitempty"`
	OrderBy           *string `url:"$orderby,omitempty"`
}

func StringRef(val string) *string {
	return &val
}

// func IntRef(val int) *int {
// 	return &val
// }

func makeMicrosoftSentinelIncidentsDataSource(clientLoader AzureClientLoadFn) *plugin.DataSource {
	return &plugin.DataSource{
		Doc:      "The `microsoft_sentinel_incidents` data source fetches incidents from Microsoft Sentinel.",
		DataFunc: fetchMicrosoftSentinelIncidents(clientLoader),
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Doc:         "The Azure client ID",
					Name:        "client_id",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
				{
					Doc:         "The Azure client secret",
					Name:        "client_secret",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
					Secret:      true,
				},
				{
					Doc:         "The Azure tenant ID",
					Name:        "tenant_id",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
				{
					Doc:         "The Azure subscription ID",
					Name:        "subscription_id",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
				{
					Doc:         "The Azure resource group name",
					Name:        "resource_group_name",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
				{
					Doc:         "The Azure workspace name",
					Name:        "workspace_name",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Doc:  "The filter expression",
					Name: "filter",
					Type: cty.String,
				},
				{
					Name:         "size",
					Doc:          "Number of objects to be returned",
					Type:         cty.Number,
					Constraints:  constraint.NonNull,
					DefaultVal:   cty.NumberIntVal(50),
					MinInclusive: cty.NumberIntVal(1),
				},
				{
					Doc:  "The order by expression",
					Name: "order_by",
					Type: cty.String,
				},
			},
		},
	}
}

func fetchMicrosoftSentinelIncidents(clientLoader AzureClientLoadFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		client, err := clientLoader(ctx, params.Config)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Unable to create Microsoft Azure client",
				Detail:   err.Error(),
			}}
		}
		req, err := prepareIncidentRequestParams(params.Config, params.Args)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse arguments",
				Detail:   err.Error(),
			}}
		}
		incidents, err := GetIncidents(ctx, client, req)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Unable to get Microsoft Sentinel incidents",
				Detail:   err.Error(),
			}}
		}
		return incidents, nil
	}
}

func prepareIncidentRequestParams(cfg, args *dataspec.Block) (*GetIncidentsReq, error) {
	var req GetIncidentsReq

	req.SubscriptionID = cfg.GetAttrVal("subscription_id").AsString()
	req.ResourceGroupName = cfg.GetAttrVal("resource_group_name").AsString()
	req.WorkspaceName = cfg.GetAttrVal("workspace_name").AsString()

	if param := args.GetAttrVal("filter"); !param.IsNull() {
		req.Filter = StringRef(param.AsString())
	}

	size64, _ := args.GetAttrVal("size").AsBigFloat().Int64()
	req.Size = int(size64)

	if param := args.GetAttrVal("order_by"); !param.IsNull() {
		req.OrderBy = StringRef(param.AsString())
	}
	return &req, nil
}

func GetIncidents(ctx context.Context, client AzureClient, req *GetIncidentsReq) (incidents plugindata.List, err error) {
	endpointTmpl := "/subscriptions/%s/resourceGroups/%s/providers/Microsoft.OperationalInsights/workspaces/%s/providers/Microsoft.SecurityInsights/incidents"
	endpoint := fmt.Sprintf(endpointTmpl, req.SubscriptionID, req.ResourceGroupName, req.WorkspaceName)

	queryParams, err := query.Values(req)
	if err != nil {
		return nil, err
	}

	incidents, err = client.QueryObjects(ctx, endpoint, queryParams, req.Size)
	return incidents, err
}
