package plugin

import (
	"fmt"
	"slices"
	"sync"

	ulid "github.com/oklog/ulid/v2"

	"github.com/blackstork-io/fabric/pkg/utils"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

type ContentID = ulid.ULID

func createId() ContentID {
	return ulid.Make()
}

type ContentKind string

const (
	SectionKind ContentKind = "section"
	EmptyKind   ContentKind = "empty"

	TextKind    ContentKind = "text"
	HeadingKind ContentKind = "heading"
	ImageKind   ContentKind = "image"
	CodeKind    ContentKind = "code"
	ListKind    ContentKind = "list"
	TOCKind     ContentKind = "toc"
)

// Content provider call result
type ContentResult struct {
	Content Content
}

type Content interface {
	ID() ContentID

	Kind() ContentKind

	AsData() plugindata.Data

	SetMeta(meta *ContentMeta)
	Meta() *ContentMeta
}

type ContentMeta struct {
	ProviderName          string
	ProviderPluginName    string
	ProviderPluginVersion string
}

func (meta *ContentMeta) AsData() plugindata.Data {
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
	id   ContentID
	meta *ContentMeta
}

func NewEmptyContent(id *ContentID, meta *ContentMeta) *ContentEmpty {
	if id == nil {
		newId := createId()
		id = &newId
	}
	return &ContentEmpty{id: *id, meta: meta}
}

func (n *ContentEmpty) SetMeta(meta *ContentMeta) {
	n.meta = meta
}

func (n *ContentEmpty) AsData() plugindata.Data {
	return plugindata.Map{
		"kind": plugindata.String(EmptyKind),
		"id":   plugindata.String(n.id.String()),
		"meta": n.meta.AsData(),
	}
}

func (n *ContentEmpty) ID() ContentID {
	return n.id
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
	id, ok := data["id"].(plugindata.String)
	if ok {
		empty.id = ContentID([]byte(id))
	}
	meta, ok := data["meta"].(plugindata.Map)
	if ok {
		empty.meta = parseContentMeta(meta)
	}
	return empty, nil
}

type ContentSection struct {
	id   ContentID
	meta *ContentMeta

	Children []Content

	mtx sync.RWMutex
}

// func NewEmptySection() *ContentSection {
// 	return &ContentSection{
// 		id: createId(),
// 	}
// }

func NewSection(id ContentID, meta *ContentMeta, children []Content) *ContentSection {
	return &ContentSection{
		id:       id,
		meta:     meta,
		Children: children,
	}
}

// Add content to the content tree.
func (c *ContentSection) Add(content Content) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.Children = append(c.Children, content)
}

func (c *ContentSection) SetMeta(meta *ContentMeta) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.meta = meta
	// FIXME: why are we propagating meta to all children?
	//
	//	for _, child := range c.Children {
	//		child.SetMeta(meta)
	//	}
}

func (c *ContentSection) ID() ContentID {
	return c.id
}

func (c *ContentSection) Meta() *ContentMeta {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.meta
}

// Compact removes empty sections from the content tree.
func (c *ContentSection) Compact() {
	c.mtx.Lock()
	defer c.mtx.Unlock()
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
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return len(c.Children) == 0
}

// AsData returns the content tree as a map.
func (c *ContentSection) AsData() plugindata.Data {
	if c == nil {
		return nil
	}
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	children := make(plugindata.List, len(c.Children))
	for i, child := range c.Children {
		children[i] = child.AsData()
	}
	return plugindata.Map{
		"id":       plugindata.String(c.id.String()),
		"kind":     plugindata.String(SectionKind),
		"children": children,
		"meta":     c.meta.AsData(),
	}
}

type ContentElement struct {
	meta *ContentMeta

	id      ContentID
	kind    ContentKind
	attrs   plugindata.Map
	dataCtx plugindata.Map

	// serialized node representation
	// 	serializedNode *astv1.FabricContentNode
	// 	source         astsrc.ASTSource
	// 	node           *nodes.FabricContentNode

	mtx sync.RWMutex
}

func NewContentElement(
	id ContentID,
	kind ContentKind,
	meta *ContentMeta,
	attrs plugindata.Map,
	data plugindata.Map,
) (*ContentElement, error) {

	elemKind := ContentKind(kind)

	if !slices.Contains([]ContentKind{
		TextKind, HeadingKind, ImageKind, CodeKind, ListKind,
	}, elemKind) {
		return nil, fmt.Errorf("unknown content type: %s", kind)
	}

	return &ContentElement{
		id:      id,
		kind:    ListKind,
		meta:    meta,
		attrs:   attrs,
		dataCtx: data,
	}, nil
}

func NewCodeElement(body string, lang string, dataCtx plugindata.Map) *ContentElement {
	return &ContentElement{
		id:   createId(),
		kind: CodeKind,
		attrs: plugindata.Map{
			"body":     plugindata.String(body),
			"language": plugindata.String(lang),
		},
		dataCtx: dataCtx,
	}
}

func NewTextElement(body string, dataCtx plugindata.Map) *ContentElement {
	return &ContentElement{
		id:   createId(),
		kind: TextKind,
		attrs: plugindata.Map{
			"body": plugindata.String(body),
		},
		dataCtx: dataCtx,
	}
}

func NewHTMLElement(body string, dataCtx plugindata.Map) *ContentElement {
	return &ContentElement{
		id:   createId(),
		kind: TextKind,
		attrs: plugindata.Map{
			"body":    plugindata.String(body),
			"is_html": plugindata.Bool(true),
		},
		dataCtx: dataCtx,
	}
}

func NewQuoteElement(body string, dataCtx plugindata.Map) *ContentElement {
	return &ContentElement{
		id:   createId(),
		kind: TextKind,
		attrs: plugindata.Map{
			"body":          plugindata.String(body),
			"is_blockquote": plugindata.Bool(true),
		},
		dataCtx: dataCtx,
	}
}

func NewHeadingElement(body string, size int64, dataCtx plugindata.Map) *ContentElement {
	return &ContentElement{
		id:   createId(),
		kind: HeadingKind,
		attrs: plugindata.Map{
			"body": plugindata.String(body),
			"size": plugindata.Number(size),
		},
		dataCtx: dataCtx,
	}
}

func NewImageElement(src string, alt string, dataCtx plugindata.Map) *ContentElement {
	return &ContentElement{
		id:   createId(),
		kind: ImageKind,
		attrs: plugindata.Map{
			"src": plugindata.String(src),
			"alt": plugindata.String(alt),
		},
		dataCtx: dataCtx,
	}
}

func NewTableElement(headers []string, cellValues plugindata.List, dataCtx plugindata.Map) *ContentElement {

	headerValues := utils.FnMap(headers, func(h string) plugindata.Data {
		return plugindata.String(h)
	})

	return &ContentElement{
		id:   createId(),
		kind: ImageKind,
		attrs: plugindata.Map{
			"headers": plugindata.List(headerValues),
			"rows":    cellValues,
		},
		dataCtx: dataCtx,
	}
}

func NewListElement(
	itemValues []plugindata.Data,
	items []string,
	format string,
	dataCtx plugindata.Map,
) *ContentElement {

	itemsData, _ := plugindata.ParseAny(items)

	return &ContentElement{
		id:   createId(),
		kind: ListKind,
		attrs: plugindata.Map{
			"items_rendered": itemsData,
			"items":          plugindata.List(itemValues),
			"format":         plugindata.String(format),
		},
		dataCtx: dataCtx,
	}
}

func NewTOCElement(
	headings plugindata.Data,
	isOrdered bool,
	dataCtx plugindata.Map,
) *ContentElement {
	return &ContentElement{
		id:   createId(),
		kind: ListKind,
		attrs: plugindata.Map{
			"headings":   headings,
			"is_ordered": plugindata.Bool(isOrdered),
		},
		dataCtx: dataCtx,
	}
}

func (c *ContentElement) ID() ContentID {
	return c.id
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

func (c *ContentElement) DataContext() plugindata.Map {
	return c.dataCtx
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

func (c *ContentElement) AsData() plugindata.Data {
	if c == nil {
		return nil
	}
	data := plugindata.Map{
		"kind":  plugindata.String(c.kind),
		"id":    plugindata.String(c.id.String()),
		"data":  c.dataCtx,
		"attrs": c.attrs,
		"meta":  c.meta.AsData(),
	}
	return data
}

func parseContentElement(kind ContentKind, data plugindata.Map) (*ContentElement, error) {

	if data == nil {
		return nil, nil
	}

	idVal, ok := data["id"].(plugindata.String)
	if !ok {
		return nil, fmt.Errorf("no id found in the content")
	}

	meta, ok := data["meta"].(plugindata.Map)
	if !ok {
		return nil, fmt.Errorf("no meta found in the content")
	}

	return NewContentElement(
		ContentID([]byte(idVal)),
		ContentKind(kind),
		parseContentMeta(meta),
		data["attrs"].(plugindata.Map),
		data["data"].(plugindata.Map),
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
	id, ok := data["id"].(plugindata.String)
	if ok {
		section.id = ContentID([]byte(id))
	}
	meta, ok := data["meta"].(plugindata.Map)
	if ok {
		section.meta = parseContentMeta(meta)
	}
	return section, nil
}
