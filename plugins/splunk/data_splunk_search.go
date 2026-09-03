// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package splunk

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugins/splunk/client"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeSplunkSearchDataSchema(loader ClientLoadFn) *plugin.DataSource {
	return &plugin.DataSource{
		Doc: "Runs a blocking Splunk search job and returns its results. Supports time bounds, result limits, status buckets, and required fields.",
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "auth_token",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
					Secret:      true,
				},
				{
					Name: "host",
					Type: cty.String,
				},
				{
					Name: "deployment_name",
					Type: cty.String,
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "search_query",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
				{
					Name: "max_count",
					Type: cty.Number,
				},
				{
					Name: "status_buckets",
					Type: cty.Number,
				},
				{
					Name: "rf",
					Type: cty.List(cty.String),
				},
				{
					Name: "earliest_time",
					Type: cty.String,
				},
				{
					Name: "latest_time",
					Type: cty.String,
				},
			},
		},
		DataFunc: fetchSplunkSearchData(loader),
	}
}

func fetchSplunkSearchData(loader ClientLoadFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		cli, err := makeClient(loader, params.Config)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to create client",
				Detail:   err.Error(),
			}}
		}

		result, err := search(cli, ctx, params.Args)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to search",
				Detail:   err.Error(),
			}}
		}
		return result, nil
	}
}

func search(cli client.Client, ctx context.Context, args *dataspec.Block) (plugindata.Data, error) {
	id, err := randID()
	if err != nil {
		return nil, err
	}
	req := &client.CreateSearchJobReq{
		ID:       id,
		ExecMode: "blocking",
	}
	if attr := args.GetAttrVal("search_query"); attr.IsNull() || attr.AsString() == "" {
		return nil, fmt.Errorf("search_query is required")
	} else {
		req.Search = attr.AsString()
	}
	if attr := args.GetAttrVal("max_count"); !attr.IsNull() {
		n, _ := attr.AsBigFloat().Int64()
		req.MaxCount = client.Int(int(n))
	}
	if attr := args.GetAttrVal("status_buckets"); !attr.IsNull() {
		n, _ := attr.AsBigFloat().Int64()
		req.StatusBuckets = client.Int(int(n))
	}
	if attr := args.GetAttrVal("rf"); !attr.IsNull() {
		req.RF = make([]string, attr.LengthInt())
		for i, v := range attr.AsValueSlice() {
			req.RF[i] = v.AsString()
		}
	}
	if attr := args.GetAttrVal("earliest_time"); !attr.IsNull() {
		req.EarliestTime = client.String(attr.AsString())
	}
	if attr := args.GetAttrVal("latest_time"); !attr.IsNull() {
		req.LatestTime = client.String(attr.AsString())
	}
	res, err := cli.CreateSearchJob(ctx, req)
	if err != nil {
		return nil, err
	}
	if res.Sid != id {
		return nil, fmt.Errorf("unexpected search job id: %s", res.Sid)
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
			res, err := cli.GetSearchJobByID(ctx, &client.GetSearchJobByIDReq{ID: id})
			if err != nil {
				return nil, err
			}
			if res.DispatchState.Failed() {
				return nil, fmt.Errorf("search job failed: %s", res.DispatchState)
			}
			if res.DispatchState.Done() {
				res, err := cli.GetSearchJobResults(ctx, &client.GetSearchJobResultsReq{
					ID:         id,
					OutputMode: "json",
				})
				if err != nil {
					return nil, err
				}
				result, err := plugindata.ParseAny(res.Results)
				if err != nil {
					return nil, err
				}
				return result, nil
			}
		}
	}
}

func randID() (string, error) {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "", err
	}
	rndStr := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	return fmt.Sprintf("fabric_%s", rndStr), nil
}
