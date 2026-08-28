// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package terraform

import (
	"context"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeTerraformStateLocalDataSource() *plugin.DataSource {
	return &plugin.DataSource{
		Config: nil,
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "path",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
			},
		},
		DataFunc: fetchTerraformStateLocalData,
	}
}

func fetchTerraformStateLocalData(
	ctx context.Context,
	params *plugin.RetrieveDataParams,
) (plugindata.Data, diagnostics.Diag) {
	path := params.Args.GetAttrVal("path")
	if path.IsNull() || path.AsString() == "" {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse arguments",
			Detail:   "path is required",
		}}
	}
	data, err := readTerraformStateFile(path.AsString())
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to read terraform state",
			Detail:   err.Error(),
		}}
	}
	return data, nil
}

func readTerraformStateFile(fp string) (plugindata.Data, error) {
	data, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	return plugindata.UnmarshalJSON(data)
}
