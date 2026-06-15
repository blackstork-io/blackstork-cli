// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package nistnvd

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugins/nistnvd/client"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

const (
	defaultLimit = 1000
	minLimit     = 1
	maxLimit     = 2000
)

func makeNistNvdCvesDataSource(loader ClientLoadFn) *plugin.DataSource {
	return &plugin.DataSource{
		DataFunc: fetchNistNvdCvesData(loader),
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:   "api_key",
					Type:   cty.String,
					Secret: true,
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "last_mod_start_date",
					Type: cty.String,
				},
				{
					Name: "last_mod_end_date",
					Type: cty.String,
				},
				{
					Name: "pub_start_date",
					Type: cty.String,
				},
				{
					Name: "pub_end_date",
					Type: cty.String,
				},
				{
					Name: "cpe_name",
					Type: cty.String,
				},
				{
					Name: "cve_id",
					Type: cty.String,
				},
				{
					Name: "cvss_v3_metrics",
					Type: cty.String,
				},
				{
					Name: "cvss_v3_severity",
					Type: cty.String,
				},
				{
					Name: "cwe_id",
					Type: cty.String,
				},
				{
					Name: "keyword_search",
					Type: cty.String,
				},
				{
					Name: "virtual_match_string",
					Type: cty.String,
				},
				{
					Name: "source_identifier",
					Type: cty.String,
				},
				{
					Name: "has_cert_alerts",
					Type: cty.Bool,
				},
				{
					Name: "has_kev",
					Type: cty.Bool,
				},
				{
					Name: "has_cert_notes",
					Type: cty.Bool,
				},
				{
					Name: "is_vulnerable",
					Type: cty.Bool,
				},
				{
					Name: "keyword_exact_match",
					Type: cty.Bool,
				},
				{
					Name: "no_rejected",
					Type: cty.Bool,
				},
				{
					Name: "limit",
					Type: cty.Number,
				},
			},
		},
	}
}

func fetchNistNvdCvesData(loader ClientLoadFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		cli, err := parseConfig(params.Config, loader)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse configuration",
			}}
		}
		req, err := parseListCVESRequest(params.Args)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse arguments",
			}}
		}
		limit := defaultLimit
		if attr := params.Args.GetAttrVal("limit"); !attr.IsNull() {
			num, _ := attr.AsBigFloat().Int64()
			limit = int(num)
			if limit < minLimit {
				limit = minLimit
			} else if limit > maxLimit {
				limit = maxLimit
			}
		}
		req.ResultsPerPage = limit
		req.StartIndex = 0
		var vulnerabilities plugindata.List
		for {
			res, err := cli.ListCVES(ctx, req)
			if err != nil {
				return nil, diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Failed to fetch data",
				}}
			}
			for _, v := range res.Vulnerabilities {
				data, err := plugindata.ParseAny(v)
				if err != nil {
					return nil, diagnostics.Diag{{
						Severity: hcl.DiagError,
						Summary:  "Failed to parse data",
					}}
				}
				vulnerabilities = append(vulnerabilities, data)
			}
			if res.StartIndex+res.ResultsPerPage >= res.TotalResults {
				break
			}
			req.StartIndex = res.StartIndex + res.ResultsPerPage
		}
		return vulnerabilities, nil
	}
}

func parseConfig(cfg *dataspec.Block, loader ClientLoadFn) (client.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}
	apiKey := cfg.GetAttrVal("api_key")
	if apiKey.IsNull() || apiKey.AsString() == "" {
		return loader(nil), nil
	}
	return loader(client.String(apiKey.AsString())), nil
}

func parseListCVESRequest(args *dataspec.Block) (*client.ListCVESReq, error) {
	if args == nil {
		return nil, fmt.Errorf("arguments are required")
	}
	req := &client.ListCVESReq{}
	if attr := args.GetAttrVal("last_mod_start_date"); !attr.IsNull() {
		req.LastModStartDate = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("last_mod_end_date"); !attr.IsNull() {
		req.LastModEndDate = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("pub_start_date"); !attr.IsNull() {
		req.PubStartDate = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("pub_end_date"); !attr.IsNull() {
		req.PubEndDate = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("cpe_name"); !attr.IsNull() {
		req.CPEName = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("cve_id"); !attr.IsNull() {
		req.CVEID = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("cvss_v3_metrics"); !attr.IsNull() {
		req.CVSSV3Metrics = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("cvss_v3_severity"); !attr.IsNull() {
		req.CVSSV3Severity = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("cwe_id"); !attr.IsNull() {
		req.CWEID = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("keyword_search"); !attr.IsNull() {
		req.KeywordSearch = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("virtual_match_string"); !attr.IsNull() {
		req.VirtualMatchString = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("source_identifier"); !attr.IsNull() {
		req.SourceIdentifier = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("has_cert_alerts"); !attr.IsNull() {
		req.HasCertAlerts = client.Bool(attr.True())
	}
	if attr := args.GetAttrVal("has_kev"); !attr.IsNull() {
		req.HasKev = client.Bool(attr.True())
	}
	if attr := args.GetAttrVal("has_cert_notes"); !attr.IsNull() {
		req.HasCertNotes = client.Bool(attr.True())
	}
	if attr := args.GetAttrVal("is_vulnerable"); !attr.IsNull() {
		req.IsVulnerable = client.Bool(attr.True())
	}
	if attr := args.GetAttrVal("keyword_exact_match"); !attr.IsNull() {
		req.KeywordExactMatch = client.Bool(attr.True())
	}
	if attr := args.GetAttrVal("no_rejected"); !attr.IsNull() {
		req.NoRejected = client.Bool(attr.True())
	}
	return req, nil
}
