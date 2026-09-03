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
	"bytes"
	"context"
	"errors"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

const (
	UnorderedListFormat = "unordered"
	OrderedListFormat   = "ordered"
)

func makeListContentProvider() *plugin.ContentProvider {
	return &plugin.ContentProvider{
		ContentFunc: genListContent,
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "item_template",
					Type:        cty.String,
					Constraints: constraint.NonNull,
					DefaultVal:  cty.StringVal("{{.}}"),
					ExampleVal:  cty.StringVal(`[{{.Title}}]({{.URL}})`),
					Doc:         "Go template for the item of the list",
				},
				{
					Name:       "format",
					Type:       cty.String,
					DefaultVal: cty.StringVal("unordered"),
					OneOf: []cty.Value{
						cty.StringVal(UnorderedListFormat),
						cty.StringVal(OrderedListFormat),
					},
				},
				{
					Name:        "items",
					Type:        cty.List(plugindata.Encapsulated.CtyType()),
					Constraints: constraint.Required,
					// Constraints: constraint.RequiredMeaningful,
					ExampleVal: cty.ListVal([]cty.Value{
						cty.StringVal("First item"),
						cty.StringVal("Second item"),
						cty.StringVal("Third item"),
					}),
					Doc: "List of items to render.",
				},
			},
		},
		Doc: "Renders a list of data items as an ordered or unordered list. Use `item_template` to control how each item appears.",
	}
}

func genListContent(
	ctx context.Context,
	params *plugin.ProvideContentParams,
) (*plugin.ContentProviderResult, error) {
	format := params.Args.GetAttrVal("format").AsString()

	items, err := utils.FnMapErr(
		params.Args.GetAttrVal("items").AsValueSlice(),
		func(v cty.Value) (plugindata.Data, error) {
			data, err := plugindata.Encapsulated.FromCty(v)
			if err != nil {
				return nil, err
			}
			if data == nil {
				return nil, nil
			}
			return *data, nil
		},
	)
	if err != nil {
		return nil, errors.Join(errors.New("failed to parse `items` argument"), err)
	}

	itemTmpl, err := template.New("item").
		Funcs(sprig.FuncMap()).
		Parse(params.Args.GetAttrVal("item_template").AsString())
	if err != nil {
		return nil, errors.Join(errors.New("failed to parse an item template"), err)
	}

	renderedItems := make([]string, len(items))

	var tmpBuf bytes.Buffer
	for i, item := range items {
		tmpBuf.Reset()
		err := itemTmpl.Execute(&tmpBuf, item.Any())
		if err != nil {
			return nil, errors.Join(errors.New("failed to render an item template"), err)
		}
		value := strings.TrimSpace(tmpBuf.String())
		renderedItems[i] = value
	}

	return &plugin.ContentProviderResult{
		Content: plugin.NewListElement(items, renderedItems, format),
	}, nil
}

// func renderListContent(format string, tmpl *template.Template, items plugindata.List) (string, error) {
// 	var buf bytes.Buffer
// 	var tmpBuf bytes.Buffer
// 	for i, item := range items {
// 		tmpBuf.Reset()
// 		err := tmpl.Execute(&tmpBuf, item.Any())
// 		if err != nil {
// 			return "", err
// 		}
// 		if format == "unordered" {
// 			buf.WriteString("* ")
// 		} else if format == "tasklist" {
// 			buf.WriteString("* [ ] ")
// 		} else {
// 			fmt.Fprintf(&buf, "%d. ", i+1)
// 		}
// 		buf.Write(bytes.TrimSpace(bytes.ReplaceAll(tmpBuf.Bytes(), []byte("\n"), []byte(" "))))
// 		buf.WriteString("\n")
// 	}
// 	return buf.String(), nil
// }
