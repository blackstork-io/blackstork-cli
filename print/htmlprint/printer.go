package htmlprint

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"

	"github.com/pelletier/go-toml/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v3"

	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/print/mdprint"
)

//go:embed document.gotempl
var documentTemplStr string
var templ = template.Must(template.New("document").Parse(documentTemplStr))

const (
	fmTitleKey       = "title"
	fmDescriptionKey = "description"
	fmCSSCodeKey     = "css_code"
	fmJSCodeKey      = "js_code"
	fmCSSSourcesKey  = "css_sources"
	fmJSSourcesKey   = "js_sources"
)

type Data struct {
	Title       string
	Description string
	Content     template.HTML
	CSS         template.CSS
	JS          template.JS
	JSSources   []template.URL
	CSSSources  []template.URL
}

// Printer is the interface for printing html content.
type Printer struct {
	md mdprint.Printer
}

// New creates a new html printer.
func New() Printer {
	return Printer{mdprint.New()}
}

// PrintString is a helper function to print html content to a string.
func PrintString(el plugin.Content) string {
	buf := bytes.NewBuffer(nil)
	if err := Print(buf, el); err != nil {
		return ""
	}
	return buf.String()
}

// Print is a helper function to print html content to a writer.
func Print(w io.Writer, el plugin.Content) error {
	p := New()
	return p.Print(context.Background(), w, el)
}

func (p Printer) Print(ctx context.Context, w io.Writer, el plugin.Content) (err error) {
	data := Data{
		Title: "Untitled",
	}
	if title, ok := p.firstTitle(el); ok {
		data.Title = title
	}
	err = p.evalFrontmatter(&data, el)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	if err := p.md.Print(ctx, buf, el); err != nil {
		return err
	}
	md := goldmark.New(
		plugin.BaseMarkdownOptions,
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)
	buff := bytes.NewBuffer(nil)
	if err := md.Convert(buf.Bytes(), buff); err != nil {
		return err
	}
	data.Content = template.HTML(buff.String()) //nolint: gosec
	return templ.Execute(w, data)
}

func (p Printer) evalFrontmatter(data *Data, el plugin.Content) error {
	fm, ok := p.extractFrontmatter(el)
	if ok {
		parsed, err := p.parseFrontmatter(fm)
		if err != nil {
			return err
		}
		if title, ok := parsed[fmTitleKey].(string); ok {
			data.Title = title
		}
		if attr, ok := parsed[fmDescriptionKey].(string); ok {
			data.Description = attr
		}
		if css, ok := parsed[fmCSSCodeKey].(string); ok {
			data.CSS = template.CSS(css) //nolint:gosec // This CSS content is trusted as it comes from frontmatter
		}
		if js, ok := parsed[fmJSCodeKey].(string); ok {
			data.JS = template.JS(js) //nolint: gosec
		}
		if sources, ok := parsed[fmCSSSourcesKey].([]any); ok {
			for _, source := range sources {
				if source, ok := source.(string); ok {
					data.CSSSources = append(data.CSSSources, template.URL(source)) //nolint: gosec
				}
			}
		}
		if sources, ok := parsed[fmJSSourcesKey].([]any); ok {
			for _, source := range sources {
				if source, ok := source.(string); ok {
					data.JSSources = append(data.JSSources, template.URL(source)) //nolint: gosec
				}
			}
		}
	}
	return nil
}

func (p Printer) firstTitle(el plugin.Content) (string, bool) {
	switch el := el.(type) {
	case *plugin.ContentSection:
		for _, c := range el.Children {
			if title, ok := p.firstTitle(c); ok {
				return title, true
			}
		}
	case *plugin.ContentElement:
		meta := el.Meta()

		if meta != nil && meta.Plugin == "blackstork/builtin" && meta.Provider == "title" {
			return string(bytes.TrimSpace(
				bytes.TrimPrefix(el.AsMarkdownSrc(), []byte("#")),
			)), true
		}
	}
	return "", false
}

func (p Printer) extractFrontmatter(el plugin.Content) (*plugin.ContentElement, bool) {
	section, ok := el.(*plugin.ContentSection)
	if !ok {
		return nil, false
	}
	for i, c := range section.Children {
		el, ok := c.(*plugin.ContentElement)
		if !ok {
			continue
		}
		meta := c.Meta()
		if meta != nil && meta.Plugin == "blackstork/builtin" && meta.Provider == "frontmatter" {
			section.Children = append(section.Children[:i], section.Children[i+1:]...)
			return el, true
		}
	}
	return nil, false
}

func (p Printer) parseFrontmatter(fm *plugin.ContentElement) (result map[string]any, err error) {
	str := fm.AsMarkdownSrc()
	switch {
	case bytes.HasPrefix(str, []byte("{")):
		err = json.Unmarshal(str, &result)
	case bytes.HasPrefix(str, []byte("---")):
		str = bytes.Trim(str, "-")
		err = yaml.Unmarshal(str, &result)
	case bytes.HasPrefix(str, []byte("+++")):
		str = bytes.Trim(str, "+")
		err = toml.Unmarshal(str, &result)
	default:
		err = fmt.Errorf("invalid frontmatter format")
	}
	return
}
