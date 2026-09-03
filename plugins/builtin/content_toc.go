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
	"slices"

	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

type providerArgs struct {
	startLevel int
	endLevel   int
	isOrdered  bool
	scope      string
}

func makeTOCContentProvider(log *slog.Logger) *plugin.ContentProvider {
	return &plugin.ContentProvider{
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:         "start_level",
					Type:         cty.Number,
					DefaultVal:   cty.NumberIntVal(0),
					Doc:          `Largest size of the header to be included the table of contents`,
					MinInclusive: cty.NumberIntVal(0),
					MaxInclusive: cty.NumberIntVal(5),
					Constraints:  constraint.Integer,
				},
				{
					Name:         "end_level",
					Type:         cty.Number,
					DefaultVal:   cty.NumberIntVal(2),
					Doc:          `Smallest size of the header to be included in the table of contents`,
					MinInclusive: cty.NumberIntVal(0),
					MaxInclusive: cty.NumberIntVal(5),
					Constraints:  constraint.Integer,
				},
				{
					Name:       "as_ordered_list",
					Type:       cty.Bool,
					DefaultVal: cty.False,
					Doc:        "Render as ordered list. If `false`, TOC is rendered as unordered list.",
				},
				{
					Name: "scope",
					Type: cty.String,
					Doc: utils.Dedent(`
					Scope for TOC to cover:
					  "document" – collect headers in the document;
					  "current" – collect headers in the current section or in the document, if TOC block is defined on the document's root level;
					`),
					OneOf: []cty.Value{
						cty.StringVal("document"),
						cty.StringVal("current"),
					},
					DefaultVal: cty.StringVal("current"),
				},
			},
		},
		// InvocationOrder: plugin.InvocationOrderEnd,
		ContentFunc: func(ctx context.Context, params *plugin.ProvideContentParams) (*plugin.ContentProviderResult, error) {
			return genTOC(ctx, log, params)
		},
		Doc: "Builds a table of contents from headings in the selected document scope. The result can be ordered or unordered and limited by heading level.",
	}
}

func parseTOCArgs(args *dataspec.Block) *providerArgs {
	startLevel, _ := args.GetAttrVal("start_level").AsBigFloat().Int64()
	endLevel, _ := args.GetAttrVal("end_level").AsBigFloat().Int64()
	ordered := args.GetAttrVal("as_ordered_list").True()
	scope := args.GetAttrVal("scope").AsString()

	return &providerArgs{
		startLevel: int(startLevel),
		endLevel:   int(endLevel),
		isOrdered:  ordered,
		scope:      scope,
	}
}

func genTOC(
	ctx context.Context,
	log *slog.Logger,
	params *plugin.ProvideContentParams,
) (*plugin.ContentProviderResult, error) {
	args := parseTOCArgs(params.Args)
	return &plugin.ContentProviderResult{
		Content: plugin.NewTOCElement(
			args.startLevel,
			args.endLevel,
			args.scope,
			args.isOrdered,
		),
	}, nil
}

func FillInTOCNodes(
	ctx context.Context,
	log *slog.Logger,
	root *plugin.ContentSection,
) (*plugin.ContentSection, error) {
	tocs, err := findTOCNodes(root)
	if err != nil {
		return nil, err
	}

	if len(tocs) == 0 {
		return root, nil
	}

	log.InfoContext(
		ctx,
		"TOC blocks found",
		"toc_blocks",
		utils.FnMap(tocs, func(toc *tocWithParent) string {
			return toc.tocElement.BlockDetails().Name
		}),
	)

	for _, toc := range tocs {

		tocBlockDetails := toc.tocElement.BlockDetails()

		_log := log.With("toc_block", tocBlockDetails.Name)

		scope := toc.tocElement.Attr("scope").(plugindata.String)

		var scopeBranch *plugin.ContentSection
		if scope == "document" {
			scopeBranch = root
		} else {
			scopeBranch = toc.parent
		}
		headings, err := findHeadingsInBranch(scopeBranch)
		if err != nil {
			_log.ErrorContext(
				ctx,
				"Error while finding headings in the branch",
				"branch",
				scopeBranch.BlockDetails().Name,
			)
			return nil, err
		}

		startLevel := int(toc.tocElement.Attr("start_level").(plugindata.Number))
		endLevel := int(toc.tocElement.Attr("end_level").(plugindata.Number))

		tocDepth := tocBlockDetails.Depth
		tocLevel := tocDepth + 1

		absoluteStartLevel := startLevel
		absoluteEndLevel := endLevel

		// if the scope if local, treat levels as relative
		if scope != "document" {
			absoluteStartLevel += tocLevel
			absoluteEndLevel += tocLevel
		}

		headingsToKeep := slices.DeleteFunc(headings, func(el *plugin.ContentElement) bool {
			levelData := el.Attr("level")

			if levelData == nil {
				return true
			}

			level := int(levelData.(plugindata.Number))
			return level < absoluteStartLevel || level > absoluteEndLevel
		})

		headingsAttrs := utils.FnMap(
			headingsToKeep,
			func(el *plugin.ContentElement) plugindata.Map {
				return el.Attrs()
			},
		)
		headingsData := make(plugindata.List, len(headingsToKeep))
		for i, v := range headingsAttrs {
			headingsData[i] = v
		}

		toc.tocElement.SetAttr("headings", plugindata.List(headingsData))

		_log.DebugContext(ctx, "TOC element filled in", "headings_count", len(headingsData))
	}

	return root, nil
}

type tocWithParent struct {
	parent     *plugin.ContentSection
	tocElement *plugin.ContentElement
}

func findTOCNodes(section *plugin.ContentSection) ([]*tocWithParent, error) {
	tocs := make([]*tocWithParent, 0)

	for _, child := range section.Children {
		switch child.Kind() {
		case plugin.TOCKind:
			tocs = append(tocs, &tocWithParent{
				parent:     section,
				tocElement: child.(*plugin.ContentElement),
			})
		case plugin.SectionKind:
			subtocs, err := findTOCNodes(child.(*plugin.ContentSection))
			if err != nil {
				return nil, err
			}
			tocs = append(tocs, subtocs...)
		}
	}

	return tocs, nil
}

func findHeadingsInBranch(section *plugin.ContentSection) ([]*plugin.ContentElement, error) {
	headings := make([]*plugin.ContentElement, 0)

	if section.Title != nil {
		el := section.Title.(*plugin.ContentElement)
		headings = append(headings, el)
	}

	for _, child := range section.Children {
		switch child.Kind() {
		case plugin.TitleKind:
			el := child.(*plugin.ContentElement)
			headings = append(headings, el)
		case plugin.SectionKind:
			subheadings, err := findHeadingsInBranch(child.(*plugin.ContentSection))
			if err != nil {
				return nil, err
			}
			headings = append(headings, subheadings...)
		}
	}

	return headings, nil
}
