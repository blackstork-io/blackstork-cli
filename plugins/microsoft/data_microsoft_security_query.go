// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package microsoft

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeMicrosoftSecurityQueryDataSource(loader MicrosoftSecurityClientLoadFn) *plugin.DataSource {
	return &plugin.DataSource{
		Doc:      "Runs a Microsoft Defender XDR advanced hunting query and returns the query results.",
		DataFunc: fetchMicrosoftSecurityQuery(loader),
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Doc:         "The Azure client ID",
					Name:        "client_id",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
				{
					Doc:    "The Azure client secret. Required if `private_key_file` or `private_key` is not provided.",
					Name:   "client_secret",
					Type:   cty.String,
					Secret: true,
				},
				{
					Doc:         "The Azure tenant ID",
					Name:        "tenant_id",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
				{
					Doc:  "The path to the private key file. Ignored if `private_key` or `client_secret` is provided.",
					Name: "private_key_file",
					Type: cty.String,
				},
				{
					Doc:  "The private key contents. Ignored if `client_secret` is provided.",
					Name: "private_key",
					Type: cty.String,
				},
				{
					Doc:  "The key passphrase. Ignored if `client_secret` is provided.",
					Name: "key_passphrase",
					Type: cty.String,
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "query",
					Doc:  "Advanced hunting query to run",
					Type: cty.String,
					ExampleVal: cty.StringVal(
						"DeviceRegistryEvents | where Timestamp >= ago(30d) | where isnotempty(RegistryKey) and isnotempty(RegistryValueName) | limit 5",
					),
					Constraints: constraint.RequiredNonNull,
				},
			},
		},
	}
}

func fetchMicrosoftSecurityQuery(loader MicrosoftSecurityClientLoadFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		cli, err := loader(ctx, params.Config)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Unable to create Microsoft Security API client",
				Detail:   err.Error(),
			}}
		}
		query := params.Args.GetAttrVal("query").AsString()

		slog.DebugContext(ctx, "Submitting an advanced hunting query", "query", query)

		response, err := cli.RunAdvancedQuery(ctx, query)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to query Microsoft Security API",
				Detail:   err.Error(),
			}}
		}
		responseMap, ok := response.(plugindata.Map)
		if !ok {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Unexpected response type received",
				Detail:   fmt.Sprintf("Unexpected object type received in the response: %T", response),
			}}
		}

		results, ok := responseMap["Results"]
		if !ok {
			slog.WarnContext(ctx, "The field `Results` is not found in the response")
			return plugindata.List{}, nil
		}

		return results, nil
	}
}
