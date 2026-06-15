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

func makeBlockQuoteContentProvider() *plugin.ContentProvider {
	return &plugin.ContentProvider{
		ContentFunc: genBlockQuoteContent,
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{{
				Name:        "value",
				Type:        cty.String,
				ExampleVal:  cty.StringVal("Text to be formatted as a quote"),
				Constraints: constraint.RequiredNonNull,
			}},
		},
		Doc: "Formats text as a blockquote",
	}
}

func genBlockQuoteContent(
	ctx context.Context,
	params *plugin.ProvideContentParams,
) (*plugin.ContentProviderResult, error) {
	value := params.Args.GetAttrVal("value")
	text, err := renderText(value.AsString(), params.DataContext)
	if err != nil {
		return nil, errors.Join(errors.New("failed to render the value as a template"), err)
	}
	// text = "> " + strings.ReplaceAll(text, "\n", "\n> ")
	return &plugin.ContentProviderResult{
		Content: plugin.NewQuoteElement(text),
	}, nil
}
