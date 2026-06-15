// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeGraphQLDataSource() *plugin.DataSource {
	return &plugin.DataSource{
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "url",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
				{
					Name:   "auth_token",
					Type:   cty.String,
					Secret: true,
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{{
				Name:        "query",
				Type:        cty.String,
				Constraints: constraint.RequiredNonNull,
			}},
		},
		DataFunc: fetchGraphQLData,
	}
}

func fetchGraphQLData(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
	url := params.Config.GetAttrVal("url")
	if url.IsNull() || url.AsString() == "" {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse config",
			Detail:   "url is required",
		}}
	}
	authToken := params.Config.GetAttrVal("auth_token")
	if authToken.IsNull() {
		authToken = cty.StringVal("")
	}
	query := params.Args.GetAttrVal("query")
	if query.IsNull() || query.AsString() == "" {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse arguments",
			Detail:   "query is required",
		}}
	}

	result, err := queryGraphQL(ctx, url.AsString(), query.AsString(), authToken.AsString())
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to execute query",
			Detail:   err.Error(),
		}}
	}

	return result, nil
}

type requestData struct {
	Query string `json:"query"`
}

func queryGraphQL(ctx context.Context, url, query, authToken string) (plugindata.Data, error) {
	data, err := json.Marshal(requestData{Query: query})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	// Set the appropriate headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	dst, err := plugindata.UnmarshalJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return dst, nil
}
