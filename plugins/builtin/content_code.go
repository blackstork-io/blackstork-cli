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
	"errors"

	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeCodeContentProvider() *plugin.ContentProvider {
	return &plugin.ContentProvider{
		ContentFunc: genCodeContent,
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "value",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
					ExampleVal:  cty.StringVal("Text to be formatted as a code block"),
				},
				{
					Name:       "language",
					Type:       cty.String,
					ExampleVal: cty.StringVal("json"),
					DefaultVal: cty.StringVal("text"),
					Doc:        `Specifiy the code language for syntax highlighting`,
				},
			},
		},
		Doc: "Renders a Go-templated string as a code block with an optional language for syntax highlighting.",
	}
}

func genCodeContent(
	ctx context.Context,
	params *plugin.ProvideContentParams,
) (*plugin.ContentProviderResult, error) {
	value := params.Args.GetAttrVal("value").AsString()
	lang := params.Args.GetAttrVal("language").AsString()
	text, err := renderText(value, params.DataContext)
	if err != nil {
		return nil, errors.Join(errors.New("failed to render code block as a template"), err)
	}
	return &plugin.ContentProviderResult{
		Content: plugin.NewCodeElement(text, lang),
	}, nil
}
