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
	"html/template"
	"io"
	"log/slog"
	"strings"

	"github.com/Masterminds/sprig/v3"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/blackstork-io/blackstork-cli/plugin"
)

const (
	documentKind      = "document"
	documentRootLevel = -1
)

func executeDynamicTemplate(ctx context.Context, w io.Writer, tmplStr string, data any) error {
	// Create a new template. You might want to cache these in a real-world scenario.
	t, err := template.New("override").Funcs(sprig.FuncMap()).Parse(tmplStr)
	if err != nil {
		// Fallback: write error to output so it's visible in the HTML
		_, _ = w.Write([]byte("\n<!-- template parse error: " + err.Error() + " -->\n"))
		return nil
	}
	return t.Execute(w, data)
}

func prerenderSectionComponents(
	ctx context.Context,
	log *slog.Logger,
	templatePerType map[TypedBlock]string,
	templatePerName map[NamedBlock]string,
	section *plugin.ContentSection,
	level int,
) (map[string]any, error) {
	data := map[string]any{
		"type": "section",
	}

	// 1. Pre-render Title
	if section.Title != nil {
		var titleBuf bytes.Buffer
		comp := ContentHTML(ctx, log, templatePerType, templatePerName, section.Title, level, nil)
		err := comp.Render(ctx, &titleBuf)
		if err != nil {
			return nil, err
		}
		data["title"] = template.HTML(titleBuf.String())

		if el, ok := section.Title.(*plugin.ContentElement); ok {
			data["title_value"] = el.Attr("value")
		}
	}

	// 2. Pre-render Children (Content)
	var contentBuf bytes.Buffer
	for _, child := range section.Children {
		// Recursive call passing the maps down
		component := ContentHTML(ctx, log, templatePerType, templatePerName, child, level+1, nil)
		err := component.Render(ctx, &contentBuf)
		if err != nil {
			return nil, err
		}
	}
	data["content"] = template.HTML(contentBuf.String())

	return data, nil
}

// prepareElementContext extracts attributes into a map.
func prepareElementContext(el plugin.Content) map[string]any {
	// If ContentElement allows iterating all attributes, do it here.
	// Otherwise, we map manually or expose a getter.
	// Assuming we construct a map of standard attributes based on Kind.

	// Simplest approach: Pass the element itself or a wrapper?
	// Standard `text/template` can't call methods with arguments easily (like .Attr("foo")).
	// We should convert known attributes to a map.

	data := make(map[string]any)

	blockDetails := el.BlockDetails()
	data["block_details"] = map[string]string{
		"id":               blockDetails.ID,
		"name":             blockDetails.Name,
		"kind":             blockDetails.Kind,
		"content_provider": blockDetails.Runner,
	}

	execDetails := el.ExecDetails()
	if execDetails != nil {
		data["exec_details"] = map[string]string{
			"content_provider": execDetails.Runner,
			"plugin_name":      execDetails.PluginName,
			"plugin_version":   execDetails.PluginVersion,
		}
	}

	switch el := el.(type) {
	case *plugin.ContentElement:
		for k, val := range el.Attrs() {
			data[k] = val.Any()
		}
	}

	return data
}

// getBlockIdentity extracts the lookup keys from the content.
// You may need to adjust "Runner" and "Name" extraction based on your specific Plugin API.
func getBlockIdentity(c plugin.Content) (TypedBlock, NamedBlock) {
	blockDetails := c.BlockDetails()
	var runner string
	if execDetails := c.ExecDetails(); execDetails != nil {
		runner = execDetails.Runner
	}
	return TypedBlock{
			kind:   blockDetails.Kind,
			runner: runner,
		}, NamedBlock{
			kind:   blockDetails.Kind,
			runner: runner,
			name:   blockDetails.Name,
		}
}

func getOverrideTemplateForBlock(
	templatePerType map[TypedBlock]string,
	templatePerName map[NamedBlock]string,
	content plugin.Content,
	level int,
) (string, bool) {
	tKey, nKey := getBlockIdentity(content)

	// Checking if the section is a document itself
	if level == documentRootLevel && content.Kind() == plugin.SectionKind {
		docTypeKey := TypedBlock{kind: documentKind}
		if tmpl, ok := templatePerType[docTypeKey]; ok {
			return tmpl, true
		}
	}

	// Check name match first, then type
	if tmpl, ok := templatePerName[nKey]; ok {
		return tmpl, true
	} else if tmpl, ok := templatePerType[tKey]; ok {
		return tmpl, true
	}
	return "", false
}

func renderMarkdownToHTML(markdown string) (string, error) {
	options := goldmark.WithExtensions(
		extension.Table,
		extension.Strikethrough,
		extension.TaskList,
	)

	md := goldmark.New(
		options,
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithXHTML(),
		),
	)
	htmlBuff := bytes.NewBuffer(nil)
	if err := md.Convert([]byte(markdown), htmlBuff); err != nil {
		return "", err
	}

	out := htmlBuff.String()
	trimmed := strings.TrimSpace(out)

	if strings.Count(trimmed, "<p>") == 1 &&
		strings.Count(trimmed, "</p>") == 1 &&
		strings.HasPrefix(trimmed, "<p>") &&
		strings.HasSuffix(trimmed, "</p>") {
		// Strip the <p> (first 3 chars) and </p> (last 4 chars)
		out = trimmed[3 : len(trimmed)-4]
	}

	return out, nil
}
