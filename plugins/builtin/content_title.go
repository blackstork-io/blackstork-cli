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
	"log/slog"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

const (
	minAbsoluteTitleSize     = 0
	maxAbsoluteTitleSize     = 5
	defaultAbsoluteTitleSize = 0
)

func makeTitleContentProvider() *plugin.ContentProvider {
	return &plugin.ContentProvider{
		ContentFunc: genTitleContent,
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "value",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
					Doc:         `Title content`,
					ExampleVal:  cty.StringVal("Vulnerability Report"),
				},
				{
					Name:        "absolute_size",
					Type:        cty.Number,
					Constraints: constraint.Integer,
					DefaultVal:  cty.NullVal(cty.Number),
					Doc: utils.Dedent(`
						Sets the absolute size of the title. If ` + "`null`" + ` – absoulute title size is determined from the document structure.
					`),
				},
				{
					Name:        "relative_size",
					Type:        cty.Number,
					Constraints: constraint.Integer,
					DefaultVal:  cty.NumberIntVal(0),
					Doc: utils.Dedent(`
						Adjusts absolute size of the title. The value (which may be negative) is added to the ` + "`absolute_size`" + ` to produce the final title size.
					`),
				},
			},
		},
		Doc: utils.Dedent(`
			Renders a Go-templated string as a heading.

			The final heading level must be from 1 through 6. Level 1 is the largest heading
			(` + "`<h1>`" + ` in HTML), and level 6 is the smallest (` + "`<h6>`" + `).
		`),
	}
}

func genTitleContent(
	ctx context.Context,
	params *plugin.ProvideContentParams,
) (*plugin.ContentProviderResult, error) {
	log := slog.Default()

	value := params.Args.GetAttrVal("value")
	if value.IsNull() {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse arguments",
			Detail:   "value is required",
		}}
	}

	// 	size := int64(0)
	// 	isRelative := false
	//
	// 	if val := params.Args.GetAttrVal("relative_size"); !val.IsNull() {
	// 		isRelative = true
	// 		size, _ = val.AsBigFloat().Int64()
	// 	}
	//
	// 	// if `absolute_size` set, it overrides `relative_size`
	// 	if val := params.Args.GetAttrVal("absolute_size"); !val.IsNull() {
	// 		isRelative = false
	// 		size, _ = val.AsBigFloat().Int64()
	// 	}

	absoluteSize := params.Args.GetAttrVal("absolute_size")
	relativeSize := params.Args.GetAttrVal("relative_size")

	details, err := plugin.ParseBlockDetails(params.DataContext[plugin.BlockDetailsDataKey])
	if err != nil {
		log.ErrorContext(ctx, "Error while parsing block details", "err", err)
		return nil, err
	}

	level := details.Depth + 1

	var size int

	if absoluteSize.IsNull() {
		size = level
	} else {
		absSize, _ := absoluteSize.AsBigFloat().Int64()
		size = int(absSize)
	}

	if !relativeSize.IsNull() {
		relSize, _ := relativeSize.AsBigFloat().Int64()
		size += int(relSize)
	}

	if size < minAbsoluteTitleSize {
		size = minAbsoluteTitleSize
	} else if size > maxAbsoluteTitleSize {
		size = maxAbsoluteTitleSize
	}

	text, err := renderText(value.AsString(), params.DataContext)
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to render title value",
			Detail:   err.Error(),
		}}
	}
	// remove all newlines
	text = strings.ReplaceAll(text, "\n", " ")

	return &plugin.ContentProviderResult{
		Content: plugin.NewHeadingElement(text, size, level),
	}, nil
}
