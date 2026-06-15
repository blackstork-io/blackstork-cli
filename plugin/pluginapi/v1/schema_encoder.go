// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package pluginapiv1

import (
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
)

func encodeSchema(src *plugin.Schema) (*Schema, diagnostics.Diag) {
	if src == nil {
		return nil, nil
	}
	var diags diagnostics.Diag

	return &Schema{
		Name:             src.Name,
		Version:          src.Version,
		DataSources:      utils.MapMapDiags(&diags, src.DataSources, encodeDataSourceSchema),
		ContentProviders: utils.MapMapDiags(&diags, src.ContentProviders, encodeContentProviderSchema),
		Formatters:       utils.MapMapDiags(&diags, src.Formatters, encodeFormatterSchema),
		Publishers:       utils.MapMapDiags(&diags, src.Publishers, encodePublisherSchema),
		Doc:              src.Doc,
		Tags:             src.Tags,
	}, diags
}

func encodeDataSourceSchema(src *plugin.DataSource) (_ *DataSourceSchema, diags diagnostics.Diag) {
	if src == nil {
		return nil, nil
	}
	schema := &DataSourceSchema{
		Doc:  src.Doc,
		Tags: src.Tags,
	}
	var diag diagnostics.Diag
	if src.Args != nil {
		schema.Args, diag = encodeRootSpec(src.Args)
		diags.Extend(diag)
	}
	if src.Config != nil {
		schema.Config, diag = encodeRootSpec(src.Config)
		diags.Extend(diag)
	}
	return schema, diags
}

func encodeContentProviderSchema(src *plugin.ContentProvider) (_ *ContentProviderSchema, diags diagnostics.Diag) {
	if src == nil {
		return nil, nil
	}
	schema := &ContentProviderSchema{
		Doc:  src.Doc,
		Tags: src.Tags,
	}
	var diag diagnostics.Diag
	if src.Args != nil {
		schema.Args, diag = encodeRootSpec(src.Args)
		diags.Extend(diag)
	}
	if src.Config != nil {
		schema.Config, diag = encodeRootSpec(src.Config)
		diags.Extend(diag)
	}
	return schema, diags
}

func encodeFormatterSchema(src *plugin.Formatter) (_ *FormatterSchema, diags diagnostics.Diag) {
	if src == nil {
		return nil, nil
	}
	schema := &FormatterSchema{
		Doc:     src.Doc,
		Format:  src.Format,
		FileExt: src.FileExt,
	}
	var diag diagnostics.Diag
	if src.Args != nil {
		schema.Args, diag = encodeRootSpec(src.Args)
		diags.Extend(diag)
	}
	if src.Config != nil {
		schema.Config, diag = encodeRootSpec(src.Config)
		diags.Extend(diag)
	}
	return schema, diags
}

func encodePublisherSchema(src *plugin.Publisher) (_ *PublisherSchema, diags diagnostics.Diag) {
	if src == nil {
		return nil, nil
	}
	schema := &PublisherSchema{
		Doc:     src.Doc,
		Tags:    src.Tags,
		Formats: src.Formats,
	}

	var diag diagnostics.Diag
	if src.Args != nil {
		schema.Args, diag = encodeRootSpec(src.Args)
		diags.Extend(diag)
	}
	if src.Config != nil {
		schema.Config, diag = encodeRootSpec(src.Config)
		diags.Extend(diag)
	}
	return schema, diags
}
