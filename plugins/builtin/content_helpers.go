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
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
)

// func getDocument(dataCtx plugindata.Map) (*plugin.ContentSection, error) {
// 	documentMap, ok := dataCtx["document"]
// 	if !ok {
// 		return nil, nil
// 	}
// 	contentMap, ok := documentMap.(plugindata.Map)["content"]
// 	if !ok {
// 		return nil, nil
// 	}
// 	return ParseContentSection(contentMap.(plugindata.Map))
// }
//
// func getRootSection(dataCtx plugindata.Map) (*plugin.ContentSection, error) {
// 	sectionMap, ok := dataCtx["section"]
// 	if !ok || sectionMap == nil {
// 		return nil, nil
// 	}
// 	contentMap, ok := sectionMap.(plugindata.Map)["content"]
// 	if !ok {
// 		return nil, nil
// 	}
// 	return ParseContentSection(contentMap.(plugindata.Map))
// }

func ParseContentSection(data plugindata.Map) (*plugin.ContentSection, error) {
	content, err := plugin.ParseContentData(data)
	if err != nil {
		return nil, err
	}
	section, ok := content.(*plugin.ContentSection)
	if !ok {
		return nil, nil
	}
	return section, nil
}

func FirstTitle(el plugin.Content) plugin.Content {
	switch el := el.(type) {
	case *plugin.ContentSection:
		if el.Title != nil {
			return el.Title
		}
		for _, c := range el.Children {
			if titleEl := FirstTitle(c); titleEl != nil {
				return titleEl
			}
		}
	case *plugin.ContentElement:
		if el.Kind() == plugin.TitleKind {
			return el
		}
	}
	return nil
}

func FirstTitleValue(el plugin.Content) *string {
	switch el := el.(type) {
	case *plugin.ContentSection:
		if el.Title != nil {
			return extractTitlePlainValue(el.Title)
		}
		for _, c := range el.Children {
			if titleEl := FirstTitle(c); titleEl != nil {
				return extractTitlePlainValue(titleEl)
			}
		}
	case *plugin.ContentElement:
		if el.Kind() == plugin.TitleKind {
			return extractTitlePlainValue(el)
		}
	}
	return nil
}

func extractTitlePlainValue(content plugin.Content) *string {
	el, ok := content.(*plugin.ContentElement)
	if !ok {
		return nil
	}

	valData := el.Attr("value")
	if valData == nil {
		return nil
	}

	val, ok := valData.(plugindata.String)
	if !ok {
		return nil
	}
	valStr := string(val)
	return &valStr
}

func prepareTemplate(name, value string) (*template.Template, error) {
	return template.New(name).Funcs(sprig.FuncMap()).Parse(value)
}

func renderText(text string, datactx plugindata.Map) (string, error) {
	tmpl, err := prepareTemplate("text", text)
	if err != nil {
		return "", fmt.Errorf("failed to parse the template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, datactx.Any())
	if err != nil {
		return "", fmt.Errorf("failed to execute the template: %w", err)
	}
	return strings.TrimSpace(buf.String()), nil
}
