// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package github

import (
	"context"
	"log/slog"

	gh "github.com/google/go-github/v84/github"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

func makeGithubGistPublisher(loader ClientLoaderFn) *plugin.Publisher {
	return &plugin.Publisher{
		Doc:  "Publishes content to github gist",
		Tags: []string{},
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{{
				Name:        "github_token",
				Type:        cty.String,
				Constraints: constraint.RequiredNonNull,
				Secret:      true,
			}},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "description",
					Type: cty.String,
				},
				{
					Name: "filename",
					Type: cty.String,
				},
				{
					Name:       "make_public",
					Type:       cty.Bool,
					DefaultVal: cty.False,
				},
				{
					Name: "gist_id",
					Type: cty.String,
				},
			},
		},
		PublishFunc: publishGithubGist(loader),
	}
}

func parseContent(data plugindata.Map) (document *plugin.ContentSection) {
	documentMap, ok := data["document"]
	if !ok {
		return document
	}
	contentMap, ok := documentMap.(plugindata.Map)["content"]
	if !ok {
		return document
	}
	content, err := plugin.ParseContentData(contentMap.(plugindata.Map))
	if err != nil {
		return document
	}
	document = content.(*plugin.ContentSection)
	return document
}

func publishGithubGist(loader ClientLoaderFn) plugin.PublishFunc {
	// TODO: confirm if to be passed from the caller
	log := slog.Default()
	// tracer := nooptrace.Tracer{}
	return func(ctx context.Context, params *plugin.PublishParams) diagnostics.Diag {
		document := parseContent(params.DataContext)
		if document == nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse the document",
				Detail:   "document is required",
			}}
		}

		log.DebugContext(ctx, "Publishing content to Github gist")

		if params.FormattedContent != nil {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "No formatted content provided",
				Detail:   "Github gist published expectes formatted content",
			}}
		}

		//		datactx := params.DataContext
		// 		datactx["format"] = plugindata.String(format)
		// 		var printer print.Printer
		// 		switch format {
		// 		case "md":
		// 			printer = mdprint.New()
		// 		case "html":
		// 			printer = htmlprint.New()
		// 		default:
		// 			return diagnostics.Diag{{
		// 				Severity: hcl.DiagError,
		// 				Summary:  "Unsupported format",
		// 				Detail:   "Only md and html formats are supported",
		// 			}}
		// 		}
		// 		printer = print.WithLogging(printer, log, slog.String("format", format))
		// 		printer = print.WithTracing(printer, tracer, attribute.String("format", format))

		// 		buff := bytes.NewBuffer(nil)
		// 		err := printer.Print(ctx, buff, document)
		// 		if err != nil {
		// 			return diagnostics.Diag{{
		// 				Severity: hcl.DiagError,
		// 				Summary:  "Failed to write to a file",
		// 				Detail:   err.Error(),
		// 			}}
		// 		}

		fileExt := params.FormattedContent.Format
		fileName := params.DocumentName + "." + fileExt
		filenameAttr := params.Args.GetAttrVal("filename")
		if !filenameAttr.IsNull() && filenameAttr.AsString() != "" {
			fileName = filenameAttr.AsString()
		}

		content := string(params.FormattedContent.Content)

		client := loader(params.Config.GetAttrVal("github_token").AsString())
		payload := &gh.Gist{
			Public: new(params.Args.GetAttrVal("make_public").True()),
			Files: map[gh.GistFilename]gh.GistFile{
				gh.GistFilename(fileName): {
					Content:  new(content),
					Filename: new(fileName),
				},
			},
		}
		// overrides params if defined
		descriptionAttr := params.Args.GetAttrVal("description")
		if !descriptionAttr.IsNull() && descriptionAttr.AsString() != "" {
			payload.Description = gh.String(descriptionAttr.AsString())
		}
		slog.InfoContext(ctx, "Publishing to GitHub gist", "filename", fileName)
		gistId := params.Args.GetAttrVal("gist_id")
		if gistId.IsNull() || gistId.AsString() == "" {
			slog.DebugContext(
				ctx,
				"No gist id set, creating a new gist",
				"is_public",
				payload.Public,
				"files",
				len(payload.Files),
			)
			gist, _, err := client.Gists().Create(ctx, payload)
			if err != nil {
				return diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Failed to create gist",
					Detail:   err.Error(),
				}}
			}
			slog.InfoContext(ctx, "Created the gist", "url", *gist.HTMLURL)
		} else {
			slog.DebugContext(ctx, "Fetching the gist", "gist_id", gistId.AsString())
			gist, _, err := client.Gists().Get(ctx, gistId.AsString())
			if err != nil {
				return diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Failed to retreive the gist",
					Detail:   err.Error(),
				}}
			}
			// changing filename or output format will create a new file instead of updating the existing one.
			// following logic will remove the old files and add new files.
			for _, file := range gist.Files {
				_, exists := payload.Files[gh.GistFilename(*file.Filename)]
				if !exists {
					payload.Files[gh.GistFilename(*file.Filename)] = gh.GistFile{}
				}
			}
			slog.DebugContext(ctx, "Updating the gist", "gist_id", gistId.AsString())
			gist, _, err = client.Gists().Edit(ctx, gistId.AsString(), payload)
			if err != nil {
				return diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Failed to update the gist",
					Detail:   err.Error(),
				}}
			}
			slog.InfoContext(ctx, "The gist updated successfully", "url", *gist.HTMLURL)
		}
		return nil
	}
}
