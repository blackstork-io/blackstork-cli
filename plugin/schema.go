package plugin

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

type Schema struct {
	Name             string
	Version          string
	Doc              string
	Tags             []string
	DataSources      DataSources
	ContentProviders ContentProviders
	Formatters       Formatters
	Publishers       Publishers
}

func (p *Schema) Validate() diagnostics.Diag {
	var diags diagnostics.Diag
	if p.Name == "" {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incomplete PluginSchema",
			Detail:   "Name not defined",
		})
	}
	if p.DataSources != nil {
		diags = append(diags, p.DataSources.Validate()...)
	}
	if p.ContentProviders != nil {
		diags = append(diags, p.ContentProviders.Validate()...)
	}
	if p.Publishers != nil {
		diags = append(diags, p.Publishers.Validate()...)
	}
	if p.DataSources == nil && p.ContentProviders == nil && p.Publishers == nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incomplete PluginSchema",
			Detail:   "No data sources, content providers or publishers defined",
		})
	}
	return diags
}

func (p *Schema) RetrieveData(
	ctx context.Context,
	name string,
	params *RetrieveDataParams,
) (_ plugindata.Data, diags diagnostics.Diag) {
	if p == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "No schema",
			Detail:   "No schema defined",
		}}
	}
	if p.DataSources == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "No data sources",
			Detail:   "No data sources defined in schema",
		}}
	}
	source, ok := p.DataSources[name]
	if !ok || source == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Data source not found",
			Detail:   fmt.Sprintf("Data source '%s' not found in schema", name),
		}}
	}
	return source.Execute(ctx, params)
}

func (p *Schema) ProvideContent(
	ctx context.Context,
	name string,
	params *ProvideContentParams,
) (_ *ContentProviderResult, diags diagnostics.Diag) {
	if p == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "No schema",
			Detail:   "No schema defined",
		}}
	}
	if p.ContentProviders == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "No content providers found",
			Detail:   "No content providers defined in schema",
		}}
	}
	provider, ok := p.ContentProviders[name]
	if !ok || provider == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Content provider not found",
			Detail:   fmt.Sprintf("Content provider '%s' not found in schema", name),
		}}
	}

	selfDetails, ok := params.DataContext["self"]
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "No self details found in data context for content",
		}}
	}
	selfName, ok := selfDetails.(plugindata.Map)["name"]
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "No self name found in data context for content",
		}}
	}

	result, diags := provider.Execute(ctx, params)
	if diags.HasErrors() {
		return nil, diags
	}

	meta, ok := params.DataContext["meta"]
	if ok {
		metaMap := meta.(plugindata.Map)
		result.Content.SetMeta(metaMap)
	}

	result.Content.SetSelf(BlockSelf{
		ProviderName:  name,
		PluginName:    p.Name,
		PluginVersion: p.Version,
		Name:          string(selfName.(plugindata.String)),
	})
	return result, diags
}

func (p *Schema) Format(
	ctx context.Context,
	name string,
	params *FormatParams,
) (_ *FormattedContent, diags diagnostics.Diag) {
	if p == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "No schema",
			Detail:   "No schema defined",
		}}
	}
	if p.Formatters == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "No formatters found",
			Detail:   "No formatters defined in schema",
		}}
	}
	formatter, ok := p.Formatters[name]
	if !ok || formatter == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Formatter not found",
			Detail:   fmt.Sprintf("Formatter '%s' not found in schema", name),
		}}
	}

	selfDetails, ok := params.DataContext["self"]
	if !ok {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "No self details found in data context for content",
		}}
	}

	result, diags := formatter.Execute(ctx, params)

	meta, ok := params.DataContext["meta"]
	if ok {
		result.Meta = meta.(plugindata.Map)
	}

	selfMap, _ := selfDetails.(plugindata.Map)
	selfName := selfMap["name"].(plugindata.String)

	result.Self = BlockSelf{
		ProviderName:  name,
		PluginName:    p.Name,
		PluginVersion: p.Version,
		Name:          string(selfName),
	}

	if diags.HasErrors() {
		return nil, diags
	}
	return result, diags
}

func (p *Schema) Publish(ctx context.Context, name string, params *PublishParams) (diags diagnostics.Diag) {
	if p == nil {
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "No schema",
			Detail:   "No schema defined",
		}}
	}
	if p.Publishers == nil {
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "No publishers found",
			Detail:   "No publishers defined in schema",
		}}
	}
	publisher, ok := p.Publishers[name]
	if !ok || publisher == nil {
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Publisher not found",
			Detail:   fmt.Sprintf("Publisher '%s' not found in schema", name),
		}}
	}
	return publisher.Execute(ctx, params)
}
