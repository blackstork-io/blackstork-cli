package plugin

import (
	"context"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

type DataSources map[string]*DataSource

func (ds DataSources) Validate() hcl.Diagnostics {
	if ds == nil {
		return nil
	}
	var diags hcl.Diagnostics
	for name, source := range ds {
		if source == nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Incomplete DataSourceSchema",
				Detail:   "DataSource '" + name + "' not loaded",
			})
		} else {
			diags = append(diags, source.Validate()...)
		}
	}
	return diags
}

type DataSource struct {
	// first non-empty line is treated as a short description
	Doc      string
	Tags     []string
	DataFunc RetrieveDataFunc
	Args     *dataspec.RootSpec
	Config   *dataspec.RootSpec
}

func (ds *DataSource) Validate() diagnostics.Diag {
	var diags diagnostics.Diag
	if ds.DataFunc == nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incomplete DataSourceSchema",
			Detail:   "DataSource function not loaded",
		})
	}
	return diags
}

func (ds *DataSource) Execute(ctx context.Context, params *RetrieveDataParams) (_ plugindata.Data, diags diagnostics.Diag) {
	if ds == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Missing DataSource schema",
		}}
	}
	diags = ds.Validate()
	if diags.HasErrors() {
		return nil, diags
	}
	return ds.DataFunc(ctx, params)
}

type RetrieveDataParams struct {
	Config *dataspec.Block
	Args   *dataspec.Block
}

type RetrieveDataFunc func(ctx context.Context, params *RetrieveDataParams) (plugindata.Data, diagnostics.Diag)
