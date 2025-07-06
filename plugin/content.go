package plugin

import (
	"errors"
	"fmt"
	"slices"

	"github.com/blackstork-io/fabric/parser/definitions"
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

	// FIXME: make data ctx key registry
	SelfDataCtxKey     = "self"
	SelfNameDataCtxKey = "name"
)

type Content interface {
	Kind() ContentKind

	AsData() plugindata.Map

	SetSelf(self BlockSelf)
	Self() BlockSelf

	SetMeta(meta plugindata.Map)
	Meta() plugindata.Map
}

type BlockSelf struct {
	PluginName    string
	PluginVersion string
	ProviderName  string

	Name string
}

func (s BlockSelf) AsData() plugindata.Map {
	return plugindata.Map{
		"provider_name":  plugindata.String(s.ProviderName),
		"plugin_name":    plugindata.String(s.PluginName),
		"plugin_version": plugindata.String(s.PluginVersion),
		"name":           plugindata.String(s.Name),
	}
}

func parseBlockSelf(data plugindata.Data) (BlockSelf, error) {
	var res BlockSelf
	if data == nil {
		return res, errors.New("no self data found")
	}
	meta := data.(plugindata.Map)
	provider, _ := meta["provider_name"].(plugindata.String)
	plugin, _ := meta["plugin_name"].(plugindata.String)
	version, _ := meta["plugin_version"].(plugindata.String)
	blockName, _ := meta["name"].(plugindata.String)
	return BlockSelf{
		PluginName:    string(plugin),
		PluginVersion: string(version),
		ProviderName:  string(provider),
		Name:          string(blockName),
	}, nil
}

type ContentEmpty struct {
	self BlockSelf
	meta plugindata.Map
}

func NewEmptyContent(self BlockSelf, meta plugindata.Map) *ContentEmpty {
	return &ContentEmpty{
		meta: meta,
		self: self,
	}
}

func (c *ContentEmpty) SetMeta(meta plugindata.Map) {
	c.meta = meta
}

func (c *ContentEmpty) SetSelf(self BlockSelf) {
	c.self = self
}

func (n *ContentEmpty) Self() BlockSelf {
	return n.self
}

func (n *ContentEmpty) AsData() plugindata.Map {
	return plugindata.Map{
		"kind":                    plugindata.String(EmptyKind),
		definitions.BlockKindMeta: n.meta,
		SelfDataCtxKey:            n.self.AsData(),
	}
}

func (n *ContentEmpty) Meta() plugindata.Map {
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
	self, ok := data[SelfDataCtxKey].(plugindata.Map)
	if ok {
		var err error
		empty.self, err = parseBlockSelf(self)
		if err != nil {
			return nil, err
		}
	}

	meta, ok := data[definitions.BlockKindMeta].(plugindata.Map)
	if ok {
		empty.meta = meta
	}

	return empty, nil
}

type ContentSection struct {
	self BlockSelf
	meta plugindata.Map

	Children []Content
}

func NewEmptySection(self BlockSelf, meta plugindata.Map) *ContentSection {
	return &ContentSection{
		meta: meta,
		self: self,
	}
}

func NewSection(self BlockSelf, meta plugindata.Map, children []Content) *ContentSection {
	return &ContentSection{
		meta:     meta,
		self:     self,
		Children: children,
	}
}

// Add content to the content tree.
func (c *ContentSection) Add(content Content) {
	c.Children = append(c.Children, content)
}

func (c *ContentSection) SetMeta(meta plugindata.Map) {
	c.meta = meta
}

func (c *ContentSection) SetSelf(self BlockSelf) {
	c.self = self
}

func (c *ContentSection) Self() BlockSelf {
	return c.self
}

func (c *ContentSection) Meta() plugindata.Map {
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
		"kind":                    plugindata.String(SectionKind),
		definitions.BlockKindMeta: c.meta,
		SelfDataCtxKey:            c.self.AsData(),
		"children":                children,
	}
}

type ContentElement struct {
	self BlockSelf
	meta plugindata.Map

	kind  ContentKind
	attrs plugindata.Map
}

func NewContentElement(
	kind ContentKind,
	self BlockSelf,
	meta plugindata.Map,
	attrs plugindata.Map,
) (*ContentElement, error) {
	return &ContentElement{
		kind:  kind,
		self:  self,
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
			"body":        plugindata.String(body),
			"size":        plugindata.Number(size),
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

func (c *ContentElement) Meta() plugindata.Map {
	return c.meta
}

func (c *ContentElement) SetMeta(meta plugindata.Map) {
	c.meta = meta
}

func (c *ContentElement) SetSelf(self BlockSelf) {
	c.self = self
}

func (c *ContentElement) Self() BlockSelf {
	return c.self
}

func (c *ContentElement) AsPluginData() plugindata.Data {
	return c.AsData()
}

func (c *ContentElement) AsData() plugindata.Map {
	if c == nil {
		return nil
	}
	data := plugindata.Map{
		"kind":                    plugindata.String(c.kind),
		SelfDataCtxKey:            c.self.AsData(),
		definitions.BlockKindMeta: c.meta,
		"attrs":                   c.attrs,
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

	self, ok := data[SelfDataCtxKey].(plugindata.Map)
	if !ok {
		return nil, fmt.Errorf("no self found in the content")
	}
	blockSelf, err := parseBlockSelf(self)
	if err != nil {
		return nil, err
	}

	meta, ok := data[definitions.BlockKindMeta].(plugindata.Map)
	if !ok {
		return nil, fmt.Errorf("no meta found in the content")
	}

	attrs := data["attrs"].(plugindata.Map)

	return NewContentElement(
		ContentKind(kind),
		blockSelf,
		meta,
		attrs,
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
	meta, ok := data[definitions.BlockKindMeta].(plugindata.Map)
	if ok {
		section.meta = meta
	}

	self, ok := data[SelfDataCtxKey].(plugindata.Map)
	if !ok {
		return nil, errors.New("Self details not found in a section data")
	}

	section.self, err = parseBlockSelf(self)
	if err != nil {
		return nil, err
	}
	return section, nil
}
