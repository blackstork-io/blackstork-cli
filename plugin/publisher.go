// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package plugin

import (
	"context"
	"slices"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

type PublishFunc func(ctx context.Context, params *PublishParams) diagnostics.Diag

type PublishParams struct {
	DocumentName string
	Config       *dataspec.Block
	Args         *dataspec.Block
	DataContext  plugindata.Map

	Content          Content
	FormattedContent *FormattedContent
}

type Publisher struct {
	Doc                string
	Tags               []string
	PublishFunc        PublishFunc
	Args               *dataspec.RootSpec
	Config             *dataspec.RootSpec
	AcceptedFormatters []string
}

func (pub *Publisher) Accepts(formatter string) bool {
	if len(pub.AcceptedFormatters) == 0 {
		return true
	}
	return slices.Contains(pub.AcceptedFormatters, formatter)
}

func (pub *Publisher) Validate() diagnostics.Diag {
	var diags diagnostics.Diag
	if pub.PublishFunc == nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incomplete Publisher schema",
			Detail:   "Publisher function not loaded",
		})
	}
	if pub.Args == nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incomplete Publisher schema",
			Detail:   "Missing args schema",
		})
	}
	return diags
}

func (pub *Publisher) Execute(ctx context.Context, params *PublishParams) (diags diagnostics.Diag) {
	if pub == nil {
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Missing Publisher schema",
		}}
	}
	if pub.PublishFunc == nil {
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Incomplete Publisher schema",
			Detail:   "Publish function not loaded",
		}}
	}
	return pub.PublishFunc(ctx, params)
}

type Publishers map[string]*Publisher

func (pubs Publishers) Validate() diagnostics.Diag {
	var diags diagnostics.Diag
	for name, provider := range pubs {
		if provider == nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Incomplete Publisher schema",
				Detail:   "Publisher '" + name + "' not loaded",
			})
		} else {
			diags = append(diags, provider.Validate()...)
		}
	}
	return diags
}
