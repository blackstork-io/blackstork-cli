package builtin

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

// func countDeclarations(data *plugin.ContentSection, name string) int {
// 	count := 0
// 	for _, child := range data.Children {
// 		if section, ok := child.(*plugin.ContentSection); ok {
// 			count += countDeclarations(section, name)
// 			continue
// 		}
// 		if element, ok := child.(*plugin.ContentElement); ok {
// 			meta := element.Meta()
// 			if meta != nil && meta.ProviderName == name {
// 				count++
// 			}
// 		}
// 	}
// 	return count
// }

func getDocument(dataCtx plugindata.Map) (*plugin.ContentSection, error) {
	documentMap, ok := dataCtx["document"]
	if !ok {
		return nil, nil
	}
	contentMap, ok := documentMap.(plugindata.Map)["content"]
	if !ok {
		return nil, nil
	}
	return parseContentSection(contentMap.(plugindata.Map))
}

func getRootSection(dataCtx plugindata.Map) (*plugin.ContentSection, error) {
	sectionMap, ok := dataCtx["section"]
	if !ok || sectionMap == nil {
		return nil, nil
	}
	contentMap, ok := sectionMap.(plugindata.Map)["content"]
	if !ok {
		return nil, nil
	}
	return parseContentSection(contentMap.(plugindata.Map))
}

func parseContentSection(data plugindata.Map) (*plugin.ContentSection, error) {
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

// func findDepth(parent *plugin.ContentSection, id plugin.ContentID, depth int) int {
// 	if parent.ID() == id {
// 		return depth
// 	}
// 	for _, child := range parent.Children {
// 		if child.ID() == id {
// 			return depth
// 		}
// 		if child, ok := child.(*plugin.ContentSection); ok {
// 			if d := findDepth(child, id, depth+1); d > -1 {
// 				return d
// 			}
// 		}
// 	}
// 	return -1
// }

func firstTitle(el plugin.Content) *plugin.ContentElement {
	switch el := el.(type) {
	case *plugin.ContentSection:
		for _, c := range el.Children {
			if titleEl := firstTitle(c); titleEl != nil {
				return titleEl
			}
		}
	case *plugin.ContentElement:
		if el.Kind() == plugin.HeadingKind {
			return el
		}
	}
	return nil
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
