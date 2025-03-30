package terraform

import (
	"context"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/plugin/plugindata"
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

func fetchTerraformStateLocalData(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
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
	data, err := os.ReadFile(fp) //nolint:gosec // File path comes from the configuration and is user-controlled
	if err != nil {
		return nil, err
	}
	return plugindata.UnmarshalJSON(data)
}
