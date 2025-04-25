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

func countDeclarations(data *plugin.ContentSection, name string) int {
	count := 0
	for _, child := range data.Children {
		if section, ok := child.(*plugin.ContentSection); ok {
			count += countDeclarations(section, name)
			continue
		}
		if element, ok := child.(*plugin.ContentElement); ok {
			meta := element.Meta()
			if meta != nil && meta.Provider == name {
				count++
			}
		}
	}
	return count
}

func parseScope(datactx plugindata.Map) (document, section *plugin.ContentSection) {
	documentMap, ok := datactx["document"]
	if !ok {
		return
	}

	contentMap, ok := documentMap.(plugindata.Map)["content"]
	if !ok {
		return
	}

	content, err := plugin.ParseContentData(contentMap.(plugindata.Map))
	if err != nil {
		return
	}

	document, ok = content.(*plugin.ContentSection)
	if !ok {
		return
	}

	sectionMap, ok := datactx["section"]
	if !ok || sectionMap == nil {
		return
	}
	contentMap, ok = sectionMap.(plugindata.Map)["content"]
	if !ok {
		return
	}
	content, err = plugin.ParseContentData(contentMap.(plugindata.Map))
	if err != nil {
		return
	}
	section, ok = content.(*plugin.ContentSection)
	if !ok {
		return
	}
	return document, section
}

func findDepth(parent *plugin.ContentSection, id plugin.ContentID, depth int) int {
	if parent.ID() == id {
		return depth
	}
	for _, child := range parent.Children {
		if child.ID() == id {
			return depth
		}
		if child, ok := child.(*plugin.ContentSection); ok {
			if d := findDepth(child, id, depth+1); d > -1 {
				return d
			}
		}
	}
	return -1
}

func firstTitle(el plugin.Content) plugin.Content {
	switch el := el.(type) {
	case *plugin.ContentSection:
		for _, c := range el.Children {
			if titleEl := firstTitle(c); titleEl != nil {
				return titleEl
			}
		}
	case *plugin.ContentElement:
		if el.Kind() == plugin.KindHeading {
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
