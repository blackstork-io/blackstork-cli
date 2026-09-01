// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

// Package plugin defines the interfaces and schemas used by BlackStork plugins.
package plugin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
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

func (p *Schema) Shortname() string {
	parts := strings.SplitN(p.Name, "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return p.Name
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
	if p.Formatters != nil {
		diags = append(diags, p.Formatters.Validate()...)
	}
	if p.Publishers != nil {
		diags = append(diags, p.Publishers.Validate()...)
	}
	if p.DataSources == nil && p.ContentProviders == nil && p.Formatters == nil && p.Publishers == nil {
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
) (_ *ContentProviderResult, err error) {
	if p == nil {
		return nil, errors.New("no schema defined")
	}
	if p.ContentProviders == nil {
		return nil, errors.New("no content providers found")
	}

	log := appctx.Log(ctx)
	log = log.With("content_provider", name)

	provider, ok := p.ContentProviders[name]
	if !ok || provider == nil {
		return nil, errors.New("content provider not found")
	}

	result, err := provider.Execute(ctx, params)
	if err != nil {
		log.ErrorContext(ctx, "Error while executing content provider", "err", err)
		return nil, err
	}

	meta, ok := params.DataContext[MetaDataKey]
	if ok {
		metaMap := meta.(plugindata.Map)
		result.Content.SetMeta(metaMap)
	}

	// set exec details on the content block
	execDetails := &ExecDetails{
		PluginName:    p.Name,
		PluginVersion: p.Version,
		Runner:        name,
	}
	result.Content.SetExecDetails(execDetails)

	// set block details on the content block
	blockDetailsData, ok := params.DataContext[BlockDetailsDataKey]
	if ok {
		blockDetails, err := ParseBlockDetails(blockDetailsData)
		if err != nil {
			log.ErrorContext(ctx, "Error while parsing block details data", "err", err)
			return nil, err
		}
		result.Content.SetBlockDetails(blockDetails)
	} else {
		log.ErrorContext(ctx, "No block details found in data context")
		return nil, errors.New("block details not found")
	}

	return result, nil
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

	result, diags := formatter.Execute(ctx, params)
	if diags.HasErrors() {
		return nil, diags
	}

	meta, ok := params.DataContext["meta"]
	if ok {
		result.Meta = meta.(plugindata.Map)
	}

	result.ExecDetails = &ExecDetails{
		Runner:        name,
		PluginName:    p.Name,
		PluginVersion: p.Version,
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
