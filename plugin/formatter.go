package plugin

import (
	"context"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

type FormatFunc func(ctx context.Context, params *FormatParams) (*FormattedContent, diagnostics.Diag)

type FormatParams struct {
	Config      *dataspec.Block
	Args        *dataspec.Block
	DataContext plugindata.Map
	Format      string
}

type Formatter struct {
	Doc        string
	Format     string
	FileExt    string
	FormatFunc FormatFunc
	Args       *dataspec.RootSpec
	Config     *dataspec.RootSpec
}

type FormattedContent struct {
	Format  string
	Content []byte
}

func (formatter *Formatter) Validate() diagnostics.Diag {
	var diags diagnostics.Diag
	if formatter.FormatFunc == nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incomplete Formatter schema",
			Detail:   "Formatter function not loaded",
		})
	}
	if formatter.Args == nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incomplete Formatter schema",
			Detail:   "Missing args schema",
		})
	}
	return diags
}

func (formatter *Formatter) Execute(
	ctx context.Context,
	params *FormatParams,
) (result *FormattedContent, diags diagnostics.Diag) {
	if formatter == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Missing Formatter schema",
		}}
	}
	if formatter.FormatFunc == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Incomplete Formatter schema",
			Detail:   "Format function not loaded",
		}}
	}
	result, diags = formatter.FormatFunc(ctx, params)
	if diags.HasErrors() {
		return nil, diags
	}
	return result, nil
}

type Formatters map[string]*Formatter

func (formatters Formatters) Validate() diagnostics.Diag {
	var diags diagnostics.Diag
	for name, formatter := range formatters {
		if formatter == nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Incomplete Formatter schema",
				Detail:   "Formatter '" + name + "' not loaded",
			})
		} else {
			diags = append(diags, formatter.Validate()...)
		}
	}
	return diags
}
