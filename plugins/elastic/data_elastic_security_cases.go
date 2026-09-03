// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package elastic

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugins/elastic/kbclient"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

const (
	minCasesSize     = 1
	defaultCasesSize = 10
)

func makeElasticSecurityCasesDataSource(loader KibanaClientLoaderFn) *plugin.DataSource {
	return &plugin.DataSource{
		Doc:      "Retrieves Elastic Security cases from Kibana. Supports filtering, searching, sorting, and limiting the returned cases.",
		DataFunc: fetchElasticSecurityCases(loader),
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "kibana_endpoint_url",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
				{
					Name:        "api_key_str",
					Type:        cty.String,
					Constraints: constraint.NonNull,
					Secret:      true,
				},
				{
					Name: "api_key",
					Type: cty.Tuple([]cty.Type{
						cty.String,
						cty.String,
					}),
					Constraints: constraint.NonNull,
					Secret:      true,
				},
			},
		},
		Args: &dataspec.RootSpec{
			Required: true,
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "space_id",
					Type: cty.String,
				},
				{
					Name: "assignees",
					Type: cty.List(cty.String),
				},
				{
					Name: "default_search_operator",
					Type: cty.String,
				},
				{
					Name: "from",
					Type: cty.String,
				},
				{
					Name: "owner",
					Type: cty.List(cty.String),
				},
				{
					Name: "reporters",
					Type: cty.List(cty.String),
				},
				{
					Name: "search",
					Type: cty.String,
				},
				{
					Name: "search_fields",
					Type: cty.List(cty.String),
				},
				{
					Name: "severity",
					Type: cty.String,
				},
				{
					Name: "sort_field",
					Type: cty.String,
				},
				{
					Name: "sort_order",
					Type: cty.String,
				},
				{
					Name: "status",
					Type: cty.String,
				},
				{
					Name: "tags",
					Type: cty.List(cty.String),
				},
				{
					Name: "to",
					Type: cty.String,
				},
				{
					Name: "size",
					Type: cty.Number,
				},
			},
		},
	}
}

func fetchElasticSecurityCases(loader KibanaClientLoaderFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		client, err := parseSecurityCasesConfig(loader, params.Config)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse configuration",
				Detail:   err.Error(),
			}}
		}
		req, err := parseSecurityCasesArgs(params.Args)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse arguments",
				Detail:   err.Error(),
			}}
		}
		size := defaultCasesSize
		if attr := params.Args.GetAttrVal("size"); !attr.IsNull() {
			num, _ := attr.AsBigFloat().Int64()
			size = int(num)
			if size < minCasesSize {
				return nil, diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Invalid size",
					Detail:   "size must be greater than 0",
				}}
			}
		}
		req.PerPage = size
		req.Page = 1
		cases := plugindata.List{}
		for {
			res, err := client.ListSecurityCases(ctx, req)
			if err != nil {
				return nil, diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Failed to list security cases",
					Detail:   err.Error(),
				}}
			}
			for _, c := range res.Cases {
				data, err := plugindata.ParseAny(c)
				if err != nil {
					return nil, diagnostics.Diag{{
						Severity: hcl.DiagError,
						Summary:  "Failed to parse security case",
						Detail:   err.Error(),
					}}
				}
				cases = append(cases, data)
			}
			if len(cases) >= size || req.Page*req.PerPage >= res.Total {
				break
			}
			req.Page++
		}
		return cases, nil
	}
}

func parseSecurityCasesArgs(args *dataspec.Block) (*kbclient.ListSecurityCasesReq, error) {
	if args == nil || len(args.Attrs) == 0 {
		return nil, fmt.Errorf("arguments are required")
	}
	req := &kbclient.ListSecurityCasesReq{}
	for name, val := range args.Attrs {
		if val.Value.IsNull() {
			continue
		}
		switch name {
		case "space_id":
			req.SpaceID = kbclient.String(val.Value.AsString())
		case "assignees":
			req.Assignees = toStringSlice(val.Value)
		case "default_search_operator":
			req.DefaultSearchOperator = kbclient.String(val.Value.AsString())
		case "from":
			req.From = kbclient.String(val.Value.AsString())
		case "owner":
			req.Owner = toStringSlice(val.Value)
		case "reporters":
			req.Reporters = toStringSlice(val.Value)
		case "search":
			req.Search = kbclient.String(val.Value.AsString())
		case "search_fields":
			req.SearchFields = toStringSlice(val.Value)
		case "severity":
			req.Severity = kbclient.String(val.Value.AsString())
		case "sort_field":
			req.SortField = kbclient.String(val.Value.AsString())
		case "sort_order":
			req.SortOrder = kbclient.String(val.Value.AsString())
		case "status":
			req.Status = kbclient.String(val.Value.AsString())
		case "tags":
			req.Tags = toStringSlice(val.Value)
		case "to":
			req.To = kbclient.String(val.Value.AsString())

		}
	}
	return req, nil
}

func toStringSlice(val cty.Value) []string {
	list := make([]string, 0, val.LengthInt())
	iter := val.ElementIterator()
	for iter.Next() {
		_, v := iter.Element()
		list = append(list, v.AsString())
	}
	return list
}

func parseSecurityCasesConfig(loader KibanaClientLoaderFn, cfg *dataspec.Block) (kbclient.Client, error) {
	var apiKey *string

	url := cfg.GetAttrVal("kibana_endpoint_url").AsString()
	if attr := cfg.GetAttrVal("api_key_str"); !attr.IsNull() {
		apiKey = kbclient.String(attr.AsString())
	} else if attr := cfg.GetAttrVal("api_key"); !attr.IsNull() {
		p1 := attr.Index(cty.NumberIntVal(0)).AsString()
		p2 := attr.Index(cty.NumberIntVal(1)).AsString()
		src := make([]byte, 0, len(p1)+len(p2)+1)
		src = append(src, p1...)
		src = append(src, ':')
		src = append(src, p2...)

		apiKey = kbclient.String(base64.RawURLEncoding.EncodeToString(src))
	}
	return loader(url, apiKey), nil
}
