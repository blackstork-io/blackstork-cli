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

func makeImageContentProvider() *plugin.ContentProvider {
	return &plugin.ContentProvider{
		ContentFunc: genImageContent,
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "src",
					Type:        cty.String,
					Constraints: constraint.RequiredMeaningful,
					ExampleVal:  cty.StringVal("https://example.com/img.png"),
				},
				{
					Name:       "alt",
					Type:       cty.String,
					ExampleVal: cty.StringVal("Text description of the image"),
					DefaultVal: cty.StringVal(""),
					// Not using empty string as DefaultVal here for semantical meaning
				},
			},
		},
		Doc: "Renders an image",
	}
}

func genImageContent(
	ctx context.Context,
	params *plugin.ProvideContentParams,
) (*plugin.ContentProviderResult, error) {
	src := params.Args.GetAttrVal("src").AsString()
	alt := params.Args.GetAttrVal("alt").AsString()

	srcStr, err := renderText(src, params.DataContext)
	if err != nil {
		return nil, errors.Join(errors.New("failed to render src value as a template"), err)
	}

	altStr, err := renderText(alt, params.DataContext)
	if err != nil {
		return nil, errors.Join(errors.New("failed to render alt value as a template"), err)
	}

	// Make sure there are no line breaks in the values
	// srcStr = strings.TrimSpace(strings.ReplaceAll(srcStr, "\n", ""))
	// altStr = strings.TrimSpace(strings.ReplaceAll(altStr, "\n", ""))
	return &plugin.ContentProviderResult{
		Content: plugin.NewImageElement(srcStr, altStr),
	}, nil
}
