// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

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

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

//go:embed stixview.gohtml
var stixViewTmplStr string

var stixViewTmpl = template.Must(template.New("stixview").Parse(stixViewTmplStr))

func makeStixViewContentProvider() *plugin.ContentProvider {
	return &plugin.ContentProvider{
		Doc: "Embeds an interactive Stixview graph for STIX objects loaded from template data, a URL, or a GitHub Gist. This provider produces HTML content.",
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

func renderStixView(ctx context.Context, params *plugin.ProvideContentParams) (*plugin.ContentProviderResult, error) {
	args, err := parseStixViewArgs(params.Args)
	if err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}
	var uid [16]byte
	_, err = rand.Read(uid[:])
	if err != nil {
		return nil, fmt.Errorf("failed to generate UID: %w", err)
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

	return &plugin.ContentProviderResult{
		Content: plugin.NewHTMLElement(buf.String()),
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

func parseStixViewArgs(args *dataspec.Block) (*stixViewArgs, error) {
	if args == nil {
		return nil, fmt.Errorf("arguments are null")
	}
	var dst stixViewArgs
	gistID := args.GetAttrVal("gist_id")
	if !gistID.IsNull() && gistID.AsString() != "" {
		dst.GistID = new(gistID.AsString())
	}
	stixURL := args.GetAttrVal("stix_url")
	if !stixURL.IsNull() && stixURL.AsString() != "" {
		dst.StixURL = new(stixURL.AsString())
	}
	caption := args.GetAttrVal("caption")
	if !caption.IsNull() && caption.AsString() != "" {
		dst.Caption = new(caption.AsString())
	}
	showFooter := args.GetAttrVal("show_footer")
	if !showFooter.IsNull() {
		dst.ShowFooter = new(showFooter.True())
	}
	showSidebar := args.GetAttrVal("show_sidebar")
	if !showSidebar.IsNull() {
		dst.ShowSidebar = new(showSidebar.True())
	}
	showTLPAsTags := args.GetAttrVal("show_tlp_as_tags")
	if !showTLPAsTags.IsNull() {
		dst.ShowTLPAsTags = new(showTLPAsTags.True())
	}
	showMarkingNodes := args.GetAttrVal("show_marking_nodes")
	if !showMarkingNodes.IsNull() {
		dst.ShowMarkingNodes = new(showMarkingNodes.True())
	}
	showLabels := args.GetAttrVal("show_labels")
	if !showLabels.IsNull() {
		dst.ShowLabels = new(showLabels.True())
	}
	showIDRefs := args.GetAttrVal("show_idrefs")
	if !showIDRefs.IsNull() {
		dst.ShowIDRefs = new(showIDRefs.True())
	}
	width := args.GetAttrVal("width")
	if !width.IsNull() {
		n, _ := width.AsBigFloat().Int64()
		dst.Width = new(int(n))
	}
	height := args.GetAttrVal("height")
	if !height.IsNull() {
		n, _ := height.AsBigFloat().Int64()
		dst.Height = new(int(n))
	}
	return &dst, nil
}
