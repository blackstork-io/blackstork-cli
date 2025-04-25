package stixview

import (
	"bytes"
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/base32"
	"fmt"
	"html/template"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

//go:embed stixview.gohtml
var stixViewTmplStr string

var stixViewTmpl = template.Must(template.New("stixview").Parse(stixViewTmplStr))

func makeStixViewContentProvider() *plugin.ContentProvider {
	return &plugin.ContentProvider{
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "gist_id",
					Type: cty.String,
				},
				{
					Name: "stix_url",
					Type: cty.String,
				},
				{
					Name: "caption",
					Type: cty.String,
				},
				{
					Name: "show_footer",
					Type: cty.Bool,
				},
				{
					Name: "show_sidebar",
					Type: cty.Bool,
				},
				{
					Name: "show_tlp_as_tags",
					Type: cty.Bool,
				},
				{
					Name: "show_marking_nodes",
					Type: cty.Bool,
				},
				{
					Name: "show_labels",
					Type: cty.Bool,
				},
				{
					Name: "show_idrefs",
					Type: cty.Bool,
				},
				{
					Name: "width",
					Type: cty.Number,
				},
				{
					Name: "height",
					Type: cty.Number,
				},
				{
					Name: "objects",
					Type: plugindata.Encapsulated.CtyType(),
				},
			},
		},
		ContentFunc: renderStixView,
	}
}

func renderStixView(ctx context.Context, params *plugin.ProvideContentParams) (*plugin.ContentResult, diagnostics.Diag) {
	args, err := parseStixViewArgs(params.Args)
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to parse arguments",
			Detail:   err.Error(),
		}}
	}
	var uid [16]byte
	_, err = rand.Read(uid[:])
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to generate UID",
			Detail:   err.Error(),
		}}
	}
	rctx := &renderContext{
		Args: args,
		UID:  base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(uid[:]),
	}

	objectCty := params.Args.GetAttrVal("objects")
	if !objectCty.IsNull() {
		objects := plugindata.Encapsulated.MustFromCty(objectCty)
		if objects != nil && *objects != nil {
			var ok bool
			rctx.Objects, ok = (*objects).(plugindata.List)
			if !ok {
				return nil, diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Invalid query result",
					Detail:   "Query result is not a list",
				}}
			}
			if rctx.Objects == nil {
				return nil, diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Invalid query result",
					Detail:   "Query result is null",
				}}
			}
		}
	}

	if rctx.Objects == nil && args.StixURL == nil && args.GistID == nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Missing arguments",
			Detail:   "Must provide either stix_url or gist_id or objects",
		}}
	}
	buf := &bytes.Buffer{}
	err = stixViewTmpl.Execute(buf, rctx)
	if err != nil {
		return nil, diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to render template",
			Detail:   err.Error(),
		}}
	}

	return &plugin.ContentResult{
		Content: plugin.NewHTMLElement(buf.String(), params.DataContext),
	}, nil
}

type renderContext struct {
	Args    *stixViewArgs
	UID     string
	Objects plugindata.List
}

type stixViewArgs struct {
	GistID           *string
	StixURL          *string
	Caption          *string
	ShowFooter       *bool
	ShowSidebar      *bool
	ShowTLPAsTags    *bool
	ShowMarkingNodes *bool
	ShowLabels       *bool
	ShowIDRefs       *bool
	Width            *int
	Height           *int
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

func parseStixViewArgs(args *dataspec.Block) (*stixViewArgs, error) {
	if args == nil {
		return nil, fmt.Errorf("arguments are null")
	}
	var dst stixViewArgs
	gistID := args.GetAttrVal("gist_id")
	if !gistID.IsNull() && gistID.AsString() != "" {
		dst.GistID = stringPtr(gistID.AsString())
	}
	stixURL := args.GetAttrVal("stix_url")
	if !stixURL.IsNull() && stixURL.AsString() != "" {
		dst.StixURL = stringPtr(stixURL.AsString())
	}
	caption := args.GetAttrVal("caption")
	if !caption.IsNull() && caption.AsString() != "" {
		dst.Caption = stringPtr(caption.AsString())
	}
	showFooter := args.GetAttrVal("show_footer")
	if !showFooter.IsNull() {
		dst.ShowFooter = boolPtr(showFooter.True())
	}
	showSidebar := args.GetAttrVal("show_sidebar")
	if !showSidebar.IsNull() {
		dst.ShowSidebar = boolPtr(showSidebar.True())
	}
	showTLPAsTags := args.GetAttrVal("show_tlp_as_tags")
	if !showTLPAsTags.IsNull() {
		dst.ShowTLPAsTags = boolPtr(showTLPAsTags.True())
	}
	showMarkingNodes := args.GetAttrVal("show_marking_nodes")
	if !showMarkingNodes.IsNull() {
		dst.ShowMarkingNodes = boolPtr(showMarkingNodes.True())
	}
	showLabels := args.GetAttrVal("show_labels")
	if !showLabels.IsNull() {
		dst.ShowLabels = boolPtr(showLabels.True())
	}
	showIDRefs := args.GetAttrVal("show_idrefs")
	if !showIDRefs.IsNull() {
		dst.ShowIDRefs = boolPtr(showIDRefs.True())
	}
	width := args.GetAttrVal("width")
	if !width.IsNull() {
		n, _ := width.AsBigFloat().Int64()
		dst.Width = intPtr(int(n))
	}
	height := args.GetAttrVal("height")
	if !height.IsNull() {
		n, _ := height.AsBigFloat().Int64()
		dst.Height = intPtr(int(n))
	}
	return &dst, nil
}
