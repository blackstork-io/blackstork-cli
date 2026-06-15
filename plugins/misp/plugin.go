// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package misp

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugins/misp/client"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

type Client interface {
	RestSearchEvents(ctx context.Context, req client.RestSearchEventsRequest) (events client.RestSearchEventsResponse, err error)
	AddEventReport(ctx context.Context, req client.AddEventReportRequest) (resp client.AddEventReportResponse, err error)
}

type ClientLoaderFn func(cfg *dataspec.Block) Client

func DefaultClientLoader(cfg *dataspec.Block) Client {
	apiKey := cfg.GetAttrVal("api_key").AsString()
	baseUrl := cfg.GetAttrVal("base_url").AsString()
	skipSsl := cfg.GetAttrVal("skip_ssl").True()
	opts := []client.ClientOption{}
	if skipSsl {
		cli := &http.Client{
			Transport: &http.Transport{
				//nolint: gosec
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
		opts = append(opts, client.WithHTTPClient(cli))
	}
	return client.NewClient(baseUrl, apiKey, opts...)
}

func Plugin(version string, loader ClientLoaderFn) *plugin.Schema {
	if loader == nil {
		loader = DefaultClientLoader
	}
	return &plugin.Schema{
		Name:    "blackstork/misp",
		Version: version,
		DataSources: plugin.DataSources{
			"misp_events": makeMispEventsDataSource(loader),
		},
		Publishers: plugin.Publishers{
			"misp_event_reports": makeMispEventReportsPublisher(loader),
		},
	}
}

// shared config for all data sources
func makeDataSourceConfig() *dataspec.RootSpec {
	return &dataspec.RootSpec{
		Attrs: []*dataspec.AttrSpec{
			{
				Name:        "api_key",
				Type:        cty.String,
				Constraints: constraint.RequiredMeaningful,
				Doc:         "misp api key",
			},
			{
				Name:        "base_url",
				Type:        cty.String,
				Constraints: constraint.RequiredMeaningful,
				Doc:         "misp base url",
			},
			{
				Name:       "skip_ssl",
				Type:       cty.Bool,
				Doc:        "skip ssl verification",
				DefaultVal: cty.BoolVal(false),
			},
		},
	}
}

func encodeResponse(data any) (plugindata.Data, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode issue: %w", err)
	}
	return plugindata.UnmarshalJSON(raw)
}
