package plugin

import (
	"fmt"
	"slices"
	"sync"

	"github.com/blackstork-io/fabric/pkg/utils"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

type ContentKind string

const (
	SectionKind ContentKind = "section"
	EmptyKind   ContentKind = "empty"

	TextKind    ContentKind = "text"
	HeadingKind ContentKind = "heading"
	ImageKind   ContentKind = "image"
	CodeKind    ContentKind = "code"
	ListKind    ContentKind = "list"
	TableKind   ContentKind = "table"
	TOCKind     ContentKind = "toc"
)

type Content interface {
	Kind() ContentKind

	AsData() plugindata.Map

	SetMeta(meta *ContentMeta)
	Meta() *ContentMeta
}

type ContentMeta struct {
	ProviderName          string
	ProviderPluginName    string
	ProviderPluginVersion string
}

func (meta *ContentMeta) AsData() plugindata.Map {
	if meta == nil {
		return nil
	}
	return plugindata.Map{
		"provider_name":           plugindata.String(meta.ProviderName),
		"provider_plugin_name":    plugindata.String(meta.ProviderPluginName),
		"provider_plugin_version": plugindata.String(meta.ProviderPluginVersion),
	}
}

func parseContentMeta(data plugindata.Data) *ContentMeta {
	if data == nil {
		return nil
	}
	meta := data.(plugindata.Map)
	provider, _ := meta["provider_name"].(plugindata.String)
	plugin, _ := meta["provider_plugin_name"].(plugindata.String)
	version, _ := meta["provider_plugin_version"].(plugindata.String)
	return &ContentMeta{
		ProviderName:          string(provider),
		ProviderPluginName:    string(plugin),
		ProviderPluginVersion: string(version),
	}
}

type ContentEmpty struct {
	meta *ContentMeta
}

func NewEmptyContent(meta *ContentMeta) *ContentEmpty {
	return &ContentEmpty{meta: meta}
}

func (n *ContentEmpty) SetMeta(meta *ContentMeta) {
	n.meta = meta
}

func (n *ContentEmpty) AsData() plugindata.Map {
	return plugindata.Map{
		"kind": plugindata.String(EmptyKind),
		"meta": n.meta.AsData(),
	}
}

func (n *ContentEmpty) Meta() *ContentMeta {
	return n.meta
}

func (c *ContentEmpty) Kind() ContentKind {
	return EmptyKind
}

func parseContentEmpty(data plugindata.Map) (*ContentEmpty, error) {
	if data == nil {
		return nil, nil
	}
	empty := &ContentEmpty{}
	meta, ok := data["meta"].(plugindata.Map)
	if ok {
		empty.meta = parseContentMeta(meta)
	}
	return empty, nil
}

type ContentSection struct {
	meta *ContentMeta

	Children []Content
}

func NewEmptySection() *ContentSection {
	return &ContentSection{}
}

func NewSection(meta *ContentMeta, children []Content) *ContentSection {
	return &ContentSection{
		meta:     meta,
		Children: children,
	}
}

// Add content to the content tree.
func (c *ContentSection) Add(content Content) {
	c.Children = append(c.Children, content)
}

func (c *ContentSection) SetMeta(meta *ContentMeta) {
	c.meta = meta
	// FIXME: why are we propagating meta to all children?
	//
	//	for _, child := range c.Children {
	//		child.SetMeta(meta)
	//	}
}

func (c *ContentSection) Meta() *ContentMeta {
	return c.meta
}

// Compact removes empty sections from the content tree.
func (c *ContentSection) Compact() {
	c.Children = slices.DeleteFunc(c.Children, func(c Content) bool {
		if _, ok := c.(*ContentEmpty); ok {
			return true
		}
		if section, ok := c.(*ContentSection); ok {
			section.Compact()
			return section.IsEmpty()
		}
		return false
	})
}

func (c *ContentSection) Kind() ContentKind {
	return SectionKind
}

// IsEmpty returns true if the section does not contain children
func (c *ContentSection) IsEmpty() bool {
	return len(c.Children) == 0
}

// AsData returns the content tree as a map.
func (c *ContentSection) AsData() plugindata.Map {
	if c == nil {
		return nil
	}
	children := make(plugindata.List, len(c.Children))
	for i, child := range c.Children {
		children[i] = child.AsData()
	}
	return plugindata.Map{
		"kind":     plugindata.String(SectionKind),
		"children": children,
		"meta":     c.meta.AsData(),
	}
}

type ContentElement struct {
	meta *ContentMeta

	kind  ContentKind
	attrs plugindata.Map

	mtx sync.RWMutex
}

func NewContentElement(
	kind ContentKind,
	meta *ContentMeta,
	attrs plugindata.Map,
) (*ContentElement, error) {
	return &ContentElement{
		kind:  kind,
		meta:  meta,
		attrs: attrs,
	}, nil
}

func NewCodeElement(body string, lang string) *ContentElement {
	return &ContentElement{
		kind: CodeKind,
		attrs: plugindata.Map{
			"body":     plugindata.String(body),
			"language": plugindata.String(lang),
		},
	}
}

func NewTextElement(body string) *ContentElement {
	return &ContentElement{
		kind: TextKind,
		attrs: plugindata.Map{
			"body": plugindata.String(body),
		},
	}
}

func NewHTMLElement(body string) *ContentElement {
	return &ContentElement{
		kind: TextKind,
		attrs: plugindata.Map{
			"body":    plugindata.String(body),
			"is_html": plugindata.Bool(true),
		},
	}
}

func NewQuoteElement(body string) *ContentElement {
	return &ContentElement{
		kind: TextKind,
		attrs: plugindata.Map{
			"body":          plugindata.String(body),
			"is_blockquote": plugindata.Bool(true),
		},
	}
}

func NewHeadingElement(body string, size int64, is_relative bool) *ContentElement {
	return &ContentElement{
		kind: HeadingKind,
		attrs: plugindata.Map{
			"body": plugindata.String(body),
			"size": plugindata.Number(size),
			"is_relative": plugindata.Bool(is_relative),
		},
	}
}

func NewImageElement(src string, alt string) *ContentElement {
	return &ContentElement{
		kind: ImageKind,
		attrs: plugindata.Map{
			"src": plugindata.String(src),
			"alt": plugindata.String(alt),
		},
	}
}

func NewTableElement(headers []string, cellValues plugindata.List) *ContentElement {

	headerValues := utils.FnMap(headers, func(h string) plugindata.Data {
		return plugindata.String(h)
	})

	return &ContentElement{
		kind: TableKind,
		attrs: plugindata.Map{
			"headers": plugindata.List(headerValues),
			"rows":    cellValues,
		},
	}
}

func NewListElement(
	itemValues []plugindata.Data,
	items []string,
	format string,
) *ContentElement {

	itemsData, _ := plugindata.ParseAny(items)

	return &ContentElement{
		kind: ListKind,
		attrs: plugindata.Map{
			"items_rendered": itemsData,
			"items":          plugindata.List(itemValues),
			"format":         plugindata.String(format),
		},
	}
}

func NewTOCElement(
	headings plugindata.Data,
	isOrdered bool,
) *ContentElement {
	return &ContentElement{
		kind: ListKind,
		attrs: plugindata.Map{
			"headings":   headings,
			"is_ordered": plugindata.Bool(isOrdered),
		},
	}
}

func (c *ContentElement) Kind() ContentKind {
	return c.kind
}

func (c *ContentElement) Attrs() plugindata.Map {
	return c.attrs
}

func (c *ContentElement) Attr(attr string) plugindata.Data {
	val, ok := c.attrs[attr]
	if !ok {
		return nil
	}
	return val
}

func (c *ContentElement) Meta() *ContentMeta {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.meta
}

func (c *ContentElement) SetMeta(meta *ContentMeta) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.meta = meta
}

func (c *ContentElement) AsPluginData() plugindata.Data {
	return c.AsData()
}

func (c *ContentElement) AsData() plugindata.Map {
	if c == nil {
		return nil
	}
	data := plugindata.Map{
		"kind":  plugindata.String(c.kind),
		"attrs": c.attrs,
		"meta":  c.meta.AsData(),
	}
	return data
}

type ContentProviderResult struct {
	Content Content
}

func parseContentElement(kind ContentKind, data plugindata.Map) (*ContentElement, error) {

	if data == nil {
		return nil, nil
	}

	meta, ok := data["meta"].(plugindata.Map)
	if !ok {
		return nil, fmt.Errorf("no meta found in the content")
	}

	return NewContentElement(
		ContentKind(kind),
		parseContentMeta(meta),
		data["attrs"].(plugindata.Map),
	)
}

func ParseContentData(data plugindata.Map) (Content, error) {
	if data == nil {
		return nil, nil
	}
	kind, ok := data["kind"].(plugindata.String)
	if !ok {
		return nil, fmt.Errorf("missing `kind` value")
	}
	switch contentKind := ContentKind(kind); contentKind {
	case SectionKind:
		return parseContentSection(data)
	case EmptyKind:
		return parseContentEmpty(data)
	default:
		return parseContentElement(contentKind, data)
	}
}

func parseContentSection(data plugindata.Map) (*ContentSection, error) {
	if data == nil {
		return nil, nil
	}
	section := &ContentSection{}
	children, ok := data["children"].(plugindata.List)
	if !ok {
		return nil, fmt.Errorf("missing content section children")
	}
	section.Children = make([]Content, len(children))
	var err error
	for i, child := range children {
		section.Children[i], err = ParseContentData(child.(plugindata.Map))
		if err != nil {
			return nil, err
		}
	}
	meta, ok := data["meta"].(plugindata.Map)
	if ok {
		section.meta = parseContentMeta(meta)
	}
	return section, nil
}
