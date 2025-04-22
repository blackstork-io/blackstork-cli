package plugin

import (
	"bytes"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	markdown "github.com/blackstork-io/goldmark-markdown"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
	"google.golang.org/protobuf/proto"

	"github.com/blackstork-io/fabric/plugin/ast/astsrc"
	"github.com/blackstork-io/fabric/plugin/ast/nodes"
	astv1 "github.com/blackstork-io/fabric/plugin/ast/v1"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

type idStore struct {
	id uint32
	sync.Mutex
}

func (i *idStore) next() uint32 {
	i.Lock()
	defer i.Unlock()
	i.id++
	return i.id
}

type LocationEffect int

const (
	LocationEffectUnspecified LocationEffect = iota
	LocationEffectBefore
	LocationEffectAfter
)

type Location struct {
	Index  uint32
	Effect LocationEffect
}

type ContentResult struct {
	Location *Location
	Content  Content
	// NewContent  *astv1.FabricContentNode
}

type Content interface {
	setID(id uint32)
	SetMeta(meta *nodes.ContentMeta)
	AsData() plugindata.Data
	ID() uint32
	AsPluginData() plugindata.Data
	Meta() *nodes.ContentMeta
}

type ContentEmpty struct {
	id   uint32
	meta *nodes.ContentMeta
}

func (n *ContentEmpty) setID(id uint32) {
	n.id = id
}

func (n *ContentEmpty) SetMeta(meta *nodes.ContentMeta) {
	n.meta = meta
}

func (n *ContentEmpty) AsData() plugindata.Data {
	return plugindata.Map{
		"type": plugindata.String("empty"),
		"id":   plugindata.Number(n.id),
		"meta": n.meta.AsData(),
	}
}

func (n *ContentEmpty) ID() uint32 {
	return n.id
}

func (n *ContentEmpty) Meta() *nodes.ContentMeta {
	return n.meta
}

func (n *ContentEmpty) AsPluginData() plugindata.Data {
	return n.AsData()
}

type ContentSection struct {
	*idStore
	id       uint32
	Children []Content
	meta     *nodes.ContentMeta
	mtx      sync.RWMutex
}

func NewSection(contentID uint32) *ContentSection {
	return &ContentSection{
		idStore: &idStore{
			id: contentID,
		},
		id: contentID,
	}
}

// Add content to the content tree.
func (c *ContentSection) Add(content Content, loc *Location) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.idStore == nil {
		c.idStore = &idStore{}
	}
	if section, ok := content.(*ContentSection); ok {
		section.idStore = c.idStore
	}
	if loc == nil {
		content.setID(c.next())
		c.Children = append(c.Children, content)
		return nil
	}
	if loc.Effect != LocationEffectUnspecified {
		content.setID(c.next())
	} else {
		content.setID(loc.Index)
	}
	foundIdx := slices.IndexFunc(c.Children, func(c Content) bool {
		return c.ID() == loc.Index
	})
	if foundIdx > -1 {
		switch loc.Effect {
		case LocationEffectBefore:
			c.Children = append(c.Children[:foundIdx], append([]Content{content}, c.Children[foundIdx:]...)...)
		case LocationEffectAfter:
			c.Children = append(c.Children[:foundIdx+1], append([]Content{content}, c.Children[foundIdx+1:]...)...)
		default:
			c.Children[foundIdx] = content
		}
		return nil
	}
	for _, c := range c.Children {
		section, ok := c.(*ContentSection)
		if !ok {
			continue
		}
		err := section.Add(content, loc)
		if err == ErrContentLocationNotFound {
			continue
		} else if err != nil {
			return err
		}
	}
	return ErrContentLocationNotFound
}

func (c *ContentSection) setID(id uint32) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.id = id
}

func (c *ContentSection) SetMeta(meta *nodes.ContentMeta) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.meta = meta
	for _, child := range c.Children {
		child.SetMeta(meta)
	}
}

func (c *ContentSection) ID() uint32 {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.id
}

func (c *ContentSection) Meta() *nodes.ContentMeta {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.meta
}

func (c *ContentSection) AsPluginData() plugindata.Data {
	return c.AsData()
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
		"type":     plugindata.String("section"),
		"id":       plugindata.Number(c.id),
		"children": children,
		"meta":     c.meta.AsData(),
	}
}

type ContentElement struct {
	// Type transitions:
	// mdString <-> source&node <-> serializedNode

	meta *nodes.ContentMeta
	id   uint32

	// do not access directly
	// legacy markdown string representation
	mdString []byte
	// serialized node representation
	serializedNode *astv1.FabricContentNode
	source         astsrc.ASTSource
	node           *nodes.FabricContentNode

	mtx sync.RWMutex
}

// NewElement is the preferred way to create a new content element.
// It accepts a list of AST nodes to build the content element.
func NewElement(content ...astv1.BlockContent) *ContentElement {
	var children []*astv1.Node
	for _, node := range content {
		children = node.ExtendNodes(children)
	}
	return &ContentElement{
		serializedNode: &astv1.FabricContentNode{
			Root: &astv1.BaseNode{
				Children: children,
			},
		},
	}
}

// NewElementFromMarkdown creates a new content element from a markdown string.
//
// Deprecated: opt in to working with the new AST by using [NewElement] instead.
func NewElementFromMarkdown(source string) *ContentElement {
	return &ContentElement{
		mdString: []byte(source),
	}
}

// NewElementFromMarkdownAndAST creates a new content element from a markdown string and an AST.
// This is a temporary method to allow for a smooth transition to the new AST.
// Should only be used for deserialization purposes during the transition.
func NewElementFromMarkdownAndAST(source []byte, ast *astv1.FabricContentNode, meta *astv1.Metadata) *ContentElement {
	return &ContentElement{
		mdString:       source,
		serializedNode: ast,
		meta:           astv1.DecodeMetadata(meta),
	}
}

var BaseMarkdownOptions = goldmark.WithExtensions(
	extension.Table,
	extension.Strikethrough,
	extension.TaskList,
)

// AsMarkdownSrc returns the markdown source of the content element.
//
// Deprecated: opt in to working with the new AST by using .AsNode()
func (c *ContentElement) AsMarkdownSrc() []byte {
	if c.mdString != nil {
		return c.mdString
	}

	source, node := c.AsNode()
	var buf bytes.Buffer
	err := goldmark.New(
		BaseMarkdownOptions,
		goldmark.WithExtensions(
			markdown.NewRenderer(
				markdown.WithIgnoredNodes(
					nodes.ContentNodeKind,
					nodes.CustomBlockKind,
					nodes.CustomInlineKind,
				),
			),
		),
	).Renderer().Render(&buf, source.AsBytes(), node)
	if err != nil {
		slog.Error("failed to render markdown", "error", err)
	}
	c.mdString = buf.Bytes()
	return c.mdString
}

func (c *ContentElement) AsSerializedNode() *astv1.FabricContentNode {
	if c.serializedNode != nil {
		return c.serializedNode
	}
	src, node := c.AsNode()
	serNode, err := astv1.Encode(node, src.AsBytes())
	if err != nil {
		slog.Error("failed to encode AST", "error", err)
	}
	c.serializedNode = serNode.GetContentNode()
	return c.serializedNode
}

func (c *ContentElement) AsNode() (*astsrc.ASTSource, *nodes.FabricContentNode) {
	if c.node != nil {
		return &c.source, c.node
	}
	if c.serializedNode != nil {
		node, source, err := astv1.Decode(&astv1.Node{
			Kind: &astv1.Node_ContentNode{
				ContentNode: c.serializedNode,
			},
		})
		if err != nil {
			slog.Error("failed to decode AST", "error", err)
		} else {
			c.node = nodes.ToFabricContentNode(node)
			c.node.Meta = c.meta
			c.source = source
		}
	} else {
		node := goldmark.New(BaseMarkdownOptions).
			Parser().Parse(text.NewReader(c.mdString))
		c.node = nodes.ToFabricContentNode(node)
		c.node.Meta = c.meta
	}

	return &c.source, c.node
}

func (c *ContentElement) ID() uint32 {
	return c.id
}

func (c *ContentElement) setID(id uint32) {
	c.id = id
}

func (c *ContentElement) Meta() *nodes.ContentMeta {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.meta
}

func (c *ContentElement) SetMeta(meta *nodes.ContentMeta) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.meta = meta
}

func (c *ContentElement) AsPluginData() plugindata.Data {
	return c.AsData()
}

func (c *ContentElement) IsAst() bool {
	return c.node != nil || c.serializedNode != nil
}

func (c *ContentElement) AsData() plugindata.Data {
	if c == nil {
		return nil
	}
	data := plugindata.Map{
		"type":     plugindata.String("element"),
		"id":       plugindata.Number(c.id),
		"markdown": plugindata.String(c.AsMarkdownSrc()),
		"meta":     c.meta.AsData(),
	}
	// we have some AST data, include it
	if c.IsAst() {
		ser, err := proto.Marshal(c.AsSerializedNode())
		if err != nil {
			slog.Warn("failed to preserve AST in element", "error", err)
		} else {
			data["__ast"] = plugindata.String(ser)
		}
	}
	return data
}

func ParseContentData(data plugindata.Map) (Content, error) {
	if data == nil {
		return nil, nil
	}
	typ, ok := data["type"].(plugindata.String)
	if !ok {
		return nil, fmt.Errorf("missing type")
	}
	switch string(typ) {
	case "section":
		return parseContentSection(data)
	case "element":
		return parseContentElement(data)
	case "empty":
		return parseContentEmpty(data)
	default:
		return nil, fmt.Errorf("unknown type: %s", typ)
	}
}

func parseContentSection(data plugindata.Map) (*ContentSection, error) {
	if data == nil {
		return nil, nil
	}
	section := &ContentSection{}
	children, ok := data["children"].(plugindata.List)
	if !ok {
		return nil, fmt.Errorf("missing children")
	}
	section.Children = make([]Content, len(children))
	var err error
	for i, child := range children {
		section.Children[i], err = ParseContentData(child.(plugindata.Map))
		if err != nil {
			return nil, err
		}
	}
	id, ok := data["id"].(plugindata.Number)
	if ok {
		section.id = uint32(id)
	}
	meta, ok := data["meta"].(plugindata.Map)
	if ok {
		section.meta = ParseContentMeta(meta)
	}
	return section, nil
}

func parseContentElement(data plugindata.Map) (*ContentElement, error) {
	if data == nil {
		return nil, nil
	}
	elem := &ContentElement{}
	markdown, ok := data["markdown"].(plugindata.String)
	if !ok {
		return nil, fmt.Errorf("missing markdown")
	}
	elem.mdString = []byte(markdown)
	id, ok := data["id"].(plugindata.Number)
	if ok {
		elem.id = uint32(id)
	}
	meta, ok := data["meta"].(plugindata.Map)
	if ok {
		elem.meta = ParseContentMeta(meta)
	}
	if astData, ok := data["__ast"].(plugindata.String); ok {
		// we have some AST data, include it
		serNode := &astv1.FabricContentNode{}
		err := proto.Unmarshal([]byte(astData), serNode)
		if err != nil {
			slog.Warn("failed to decode AST in element", "error", err)
		} else {
			elem.serializedNode = serNode
		}
	}
	return elem, nil
}

func parseContentEmpty(data plugindata.Map) (*ContentEmpty, error) {
	if data == nil {
		return nil, nil
	}
	empty := &ContentEmpty{}
	id, ok := data["id"].(plugindata.Number)
	if !ok {
		return nil, fmt.Errorf("missing id")
	}
	empty.id = uint32(id)
	meta, ok := data["meta"].(plugindata.Map)
	if ok {
		empty.meta = ParseContentMeta(meta)
	}
	return empty, nil
}

func ParseContentMeta(data plugindata.Data) *nodes.ContentMeta {
	if data == nil {
		return nil
	}
	meta := data.(plugindata.Map)
	provider, _ := meta["provider"].(plugindata.String)
	plugin, _ := meta["plugin"].(plugindata.String)
	version, _ := meta["version"].(plugindata.String)
	return &nodes.ContentMeta{
		Provider: string(provider),
		Plugin:   string(plugin),
		Version:  string(version),
	}
}
