// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package builtin

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeTextContentProvider() *plugin.ContentProvider {
	return &plugin.ContentProvider{
		ContentFunc: genTextContent,
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "value",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
					ExampleVal:  cty.StringVal("Hello world!"),
					Doc:         `Text value rendered as a Go template`,
				},
			},
		},
		Doc: `Renders text block`,
	}
}

func genTextContent(
	ctx context.Context,
	params *plugin.ProvideContentParams,
) (*plugin.ContentProviderResult, error) {
	value := params.Args.GetAttrVal("value")
	if value.IsNull() {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse the arguments",
			Detail:   "`value` must be provided",
		}}
	}

	text, err := renderText(value.AsString(), params.DataContext)
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to render text block",
			Detail:   err.Error(),
		}}
	}
	return &plugin.ContentProviderResult{
		Content: plugin.NewTextElement(text),
	}, nil
}
