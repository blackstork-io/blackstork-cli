package builtin

import (
	"context"
	"log/slog"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/utils"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

type providerArgs struct {
	startLevel int
	endLevel   int
	isOrdered  bool
	scope      string
}

type heading struct {
	title string
	level int
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
					  "section" – collect headers in the current section;
					  "auto" – adaptive behaviour, with "section" scope if the toc" block is defined inside of a section,
					  and "document" if it's on the root level of the document.
					`),
					OneOf: []cty.Value{
						cty.StringVal("document"),
						cty.StringVal("section"),
						cty.StringVal("auto"),
					},
					DefaultVal: cty.StringVal("auto"),
				},
			},
		},
		//InvocationOrder: plugin.InvocationOrderEnd,
		ContentFunc: func(ctx context.Context, params *plugin.ProvideContentParams) (*plugin.ContentProviderResult, diagnostics.Diag) {
			return genTOC(ctx, log, params)
		},
		Doc: `Renders a list of contents (TOC) from the headers found in a defined scope.`,
	}
}

func parseTOCArgs(args *dataspec.Block) (*providerArgs, error) {
	startLevel, _ := args.GetAttrVal("start_level").AsBigFloat().Int64()
	endLevel, _ := args.GetAttrVal("end_level").AsBigFloat().Int64()
	ordered := args.GetAttrVal("as_ordered_list").True()
	scope := args.GetAttrVal("scope").AsString()

	return &providerArgs{
		startLevel: int(startLevel),
		endLevel:   int(endLevel),
		isOrdered:  ordered,
		scope:      scope,
	}, nil
}

func genTOC(
	ctx context.Context,
	log *slog.Logger,
	params *plugin.ProvideContentParams,
) (*plugin.ContentProviderResult, diagnostics.Diag) {
	args, err := parseTOCArgs(params.Args)
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse the arguments",
			Detail:   err.Error(),
		}}
	}
	headings := []heading{}
	// headings, err := extractContentTitles(log, params.DataContext, args.startLevel, args.endLevel, args.scope)
	// if err != nil {
	// 	return nil, diagnostics.Diag{{
	// 		Severity: hcl.DiagError,
	// 		Summary:  "Failed to find the headings in the content",
	// 		Detail:   err.Error(),
	// 	}}
	// }

	headingsAsData := plugindata.List(utils.FnMap(headings, func(h heading) plugindata.Data {
		return plugindata.Map{
			"title": plugindata.String(h.title),
			"level": plugindata.Number(h.level),
		}
	}))

	return &plugin.ContentProviderResult{
		Content: plugin.NewTOCElement(headingsAsData, args.isOrdered),
	}, nil
}

// func (n tocNode) render(pos, depth int, ordered bool) string {
// 	format := "%s- [%s](#%s)\n"
// 	if ordered {
// 		format = "%s" + strconv.Itoa(pos+1) + ". [%s](#%s)\n"
// 	}
// 	const indentStep = "  "
// 	dst := []string{
// 		fmt.Sprintf(format, strings.Repeat(indentStep, depth), n.title, anchorize(n.title)),
// 		n.children.render(depth+1, ordered),
// 	}
// 	return strings.Join(dst, "")
// }

// func (l tocNodeList) render(depth int, ordered bool) string {
// 	dst := []string{}
// 	for i, node := range l {
// 		dst = append(dst, node.render(i, depth, ordered))
// 	}
// 	return strings.Join(dst, "")
// }
//
// func (l tocNodeList) add(node tocNode) tocNodeList {
// 	if len(l) == 0 {
// 		return append(l, node)
// 	}
// 	last := l[len(l)-1]
// 	if last.level < node.level {
// 		last.children = last.children.add(node)
// 		l[len(l)-1] = last
// 	} else {
// 		l = append(l, node)
// 	}
// 	return l
// }
//
// func anchorize(s string) string {
// 	return strings.ToLower(strings.ReplaceAll(s, " ", "-"))
// }

func findTitles(log *slog.Logger, section *plugin.ContentSection) []heading {

	headings := []heading{}

	// Depth-first recursive walk

	for _, content := range section.Children {
		switch content := content.(type) {
		case *plugin.ContentSection:
			// Collect child headings in the sub section
			headings = append(headings, findTitles(log, content)...)
		case *plugin.ContentElement:
			if content.Kind() != plugin.HeadingKind {
				continue
			}

			title := content.Attr("body")
			size := content.Attr("size")

			// Skip invalid title blocks
			if title == nil || size == nil {
				log.Error("Invalid title element encountered, skipping", "title", title, "size", size)
				continue
			}
			heading := heading{
				title: string(title.(plugindata.String)),
				level: int(size.(plugindata.Number)),
			}
			headings = append(headings, heading)
		}
	}

	return headings
}

// func extractContentTitles(
// 	log *slog.Logger,
// 	data plugindata.Map,
// 	startLvl, endLvl int,
// 	scope string,
// ) (headings []heading, err error) {
//
// 	document, err := getDocument(data)
//
// 	section, err := getRootSection(data)
//
// 	if scope == "auto" && section != nil {
// 		scope = "section"
// 	} else if scope == "auto" && section == nil {
// 		scope = "document"
// 	}
//
// 	if scope == "document" {
// 		headings = findTitles(log, document)
// 	} else if scope == "section" && section != nil {
// 		headings = findTitles(log, section)
// 	} else {
// 		return nil, fmt.Errorf("no content in the scope")
// 	}
//
// 	headings = slices.DeleteFunc(headings, func(h heading) bool {
// 		toKeep := (startLvl <= h.level) && (h.level <= endLvl)
// 		return !toKeep
// 	})
// 	return headings, nil
// }
