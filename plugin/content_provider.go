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
	"errors"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

type ContentProviders map[string]*ContentProvider

func (cp ContentProviders) Validate() diagnostics.Diag {
	var diags diagnostics.Diag
	for name, provider := range cp {
		if provider == nil {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Incomplete ContentProviderSchema",
				Detail:   "ContentProvider '" + name + "' not loaded",
			})
		} else {
			diags = append(diags, provider.Validate()...)
		}
	}
	return diags
}

type ContentProvider struct {
	// first non-empty line is treated as a short description
	Doc         string
	Tags        []string
	ContentFunc ProvideContentFunc
	Args        *dataspec.RootSpec
	Config      *dataspec.RootSpec
}

func (p *ContentProvider) Validate() diagnostics.Diag {
	var diags diagnostics.Diag
	if p.ContentFunc == nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incomplete ContentProviderSchema",
			Detail:   "ContentProvider function not loaded",
		})
	}
	if p.Args == nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incomplete ContentProviderSchema",
			Detail:   "Missing args schema",
		})
	}
	return diags
}

func (p *ContentProvider) Execute(
	ctx context.Context,
	params *ProvideContentParams,
) (_ *ContentProviderResult, err error) {
	if p == nil {
		return nil, errors.New("missing content provider schema")
	}
	if p.ContentFunc == nil {
		return nil, errors.New("content provider function not set")
	}
	return p.ContentFunc(ctx, params)
}

type ProvideContentParams struct {
	Config      *dataspec.Block
	Args        *dataspec.Block
	DataContext plugindata.Map
}

type ProvideContentFunc func(ctx context.Context, params *ProvideContentParams) (*ContentProviderResult, error)
