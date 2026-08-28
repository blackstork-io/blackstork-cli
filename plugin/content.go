// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package plugin

import (
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

type ContentKind string

const (
	SectionKind    ContentKind = "section"
	EmptyKind      ContentKind = "empty"
	TitleKind      ContentKind = "title"
	TextKind       ContentKind = "text"
	HTMLKind       ContentKind = "html"
	BlockquoteKind ContentKind = "blockquote"
	ImageKind      ContentKind = "image"
	CodeKind       ContentKind = "code"
	ListKind       ContentKind = "list"
	TableKind      ContentKind = "table"
	TOCKind        ContentKind = "toc"

	DataDataKey     = "data"
	DocumentDataKey = "document"

	ContentIDDataKey       = "id"
	ContentKindDataKey     = "kind"
	ContentChildrenDataKey = "children"
	ContentTitleDataKey    = "title"

	BlockDetailsDataKey = "block_details"
	ExecDetailsDataKey  = "exec_details"

	InputsDataKey = "inputs"
	MetaDataKey   = "meta"
	AttrsDataKey  = "attrs"

	DependenciesDataKey = "deps"
)

type Content interface {
	ID() uuid.UUID
	Kind() ContentKind

	AsData() plugindata.Map

	SetMeta(meta plugindata.Map)
	Meta() plugindata.Map

	BlockDetails() *BlockDetails
	SetBlockDetails(*BlockDetails)

	ExecDetails() *ExecDetails
	SetExecDetails(*ExecDetails)
}

type ContentSection struct {
	id   uuid.UUID
	meta plugindata.Map

	blockDetails *BlockDetails

	Title Content

	Children []Content
}

func NewEmptySection(details *BlockDetails, meta plugindata.Map) *ContentSection {
	return &ContentSection{
		id:           newContentID(),
		blockDetails: details,
		meta:         meta,
	}
}

func NewSection(
	id uuid.UUID,
	blockDetails *BlockDetails,
	meta plugindata.Map,
	title Content,
	children []Content,
) *ContentSection {
	return &ContentSection{
		id:           id,
		blockDetails: blockDetails,
		meta:         meta,
		Title:        title,
		Children:     children,
	}
}

// Add content to the content tree.
func (c *ContentSection) Add(content Content) {
	c.Children = append(c.Children, content)
}

func (c *ContentSection) SetMeta(meta plugindata.Map) {
	c.meta = meta
}

func (c *ContentSection) ID() uuid.UUID {
	return c.id
}

func (c *ContentSection) BlockDetails() *BlockDetails {
	return c.blockDetails
}

func (c *ContentSection) SetBlockDetails(blockDetails *BlockDetails) {
	c.blockDetails = blockDetails
}

func (c *ContentSection) ExecDetails() *ExecDetails {
	return nil
}

func (c *ContentSection) SetExecDetails(_ *ExecDetails) {}

func (c *ContentSection) Meta() plugindata.Map {
	return c.meta
}

// Compact removes empty sections from the content tree.
func (c *ContentSection) Compact() {
	c.Children = slices.DeleteFunc(c.Children, func(c Content) bool {
		if c.Kind() == EmptyKind {
			return true
		}
		if c.Kind() == SectionKind {
			section := c.(*ContentSection)
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
	return c.Title == nil && len(c.Children) == 0
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

	var title plugindata.Map
	if c.Title != nil {
		title = c.Title.AsData()
	}

	return plugindata.Map{
		ContentIDDataKey:       plugindata.String(c.id.String()),
		ContentKindDataKey:     plugindata.String(SectionKind),
		ContentChildrenDataKey: children,
		ContentTitleDataKey:    title,
		MetaDataKey:            c.meta,
		BlockDetailsDataKey:    c.blockDetails.AsData(),
	}
}

type ContentElement struct {
	id           uuid.UUID
	kind         ContentKind
	blockDetails *BlockDetails
	execDetails  *ExecDetails
	meta         plugindata.Map
	attrs        plugindata.Map
}

func NewContentElement(
	id uuid.UUID,
	kind ContentKind,
	blockDetails *BlockDetails,
	execDetails *ExecDetails,
	meta plugindata.Map,
	attrs plugindata.Map,
) *ContentElement {
	return &ContentElement{
		id:           id,
		kind:         kind,
		blockDetails: blockDetails,
		execDetails:  execDetails,
		meta:         meta,
		attrs:        attrs,
	}
}

func NewCodeElement(body, lang string) *ContentElement {
	return &ContentElement{
		id:   newContentID(),
		kind: CodeKind,
		attrs: plugindata.Map{
			"value":    plugindata.String(body),
			"language": plugindata.String(lang),
		},
	}
}

func NewTextElement(body string) *ContentElement {
	return &ContentElement{
		id:   newContentID(),
		kind: TextKind,
		attrs: plugindata.Map{
			"value": plugindata.String(body),
		},
	}
}

func NewHTMLElement(body string) *ContentElement {
	return &ContentElement{
		id:   newContentID(),
		kind: HTMLKind,
		attrs: plugindata.Map{
			"value": plugindata.String(body),
		},
	}
}

func NewQuoteElement(body string) *ContentElement {
	return &ContentElement{
		id:   newContentID(),
		kind: BlockquoteKind,
		attrs: plugindata.Map{
			"value": plugindata.String(body),
		},
	}
}

func NewHeadingElement(body string, size, level int) *ContentElement {
	return &ContentElement{
		id:   newContentID(),
		kind: TitleKind,
		attrs: plugindata.Map{
			"value": plugindata.String(body),
			"size":  plugindata.Number(size),
			"level": plugindata.Number(level),
		},
	}
}

func NewImageElement(src, alt string) *ContentElement {
	return &ContentElement{
		id:   newContentID(),
		kind: ImageKind,
		attrs: plugindata.Map{
			"src": plugindata.String(src),
			"alt": plugindata.String(alt),
		},
	}
}

func NewTableElement(headers, cellValues plugindata.List) *ContentElement {
	return &ContentElement{
		id:   newContentID(),
		kind: TableKind,
		attrs: plugindata.Map{
			"headers": headers,
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
		id:   newContentID(),
		kind: ListKind,
		attrs: plugindata.Map{
			"items_rendered": itemsData,
			"items":          plugindata.List(itemValues),
			"format":         plugindata.String(format),
		},
	}
}

func NewTOCElement(
	startLevel int,
	endLevel int,
	scope string,
	isOrdered bool,
) *ContentElement {
	return &ContentElement{
		id:   newContentID(),
		kind: TOCKind,
		attrs: plugindata.Map{
			"start_level": plugindata.Number(startLevel),
			"end_level":   plugindata.Number(endLevel),
			"scope":       plugindata.String(scope),
			"is_ordered":  plugindata.Bool(isOrdered),
			"headings":    plugindata.List{}, // a list of attrs from TitleKind elements, filled in in post-execution

			// "value":        plugindata.String
			// "size":        plugindata.Number
			// "depth":       plugindata.Number
		},
	}
}

func (c *ContentElement) ID() uuid.UUID {
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

func (c *ContentElement) SetAttr(attr string, value plugindata.Data) {
	c.attrs[attr] = value
}

func (c *ContentElement) Meta() plugindata.Map {
	return c.meta
}

func (c *ContentElement) SetMeta(meta plugindata.Map) {
	c.meta = meta
}

func (c *ContentElement) BlockDetails() *BlockDetails {
	return c.blockDetails
}

func (c *ContentElement) SetBlockDetails(blockDetails *BlockDetails) {
	c.blockDetails = blockDetails
}

func (c *ContentElement) ExecDetails() *ExecDetails {
	return c.execDetails
}

func (c *ContentElement) SetExecDetails(execDetails *ExecDetails) {
	c.execDetails = execDetails
}

func (c *ContentElement) AsPluginData() plugindata.Data {
	return c.AsData()
}

func (c *ContentElement) AsData() plugindata.Map {
	if c == nil {
		return nil
	}
	data := plugindata.Map{
		ContentIDDataKey:          plugindata.String(c.id.String()),
		ContentKindDataKey:        plugindata.String(c.kind),
		definitions.BlockKindMeta: c.meta,
		AttrsDataKey:              c.attrs,
	}

	if c.blockDetails != nil {
		data[BlockDetailsDataKey] = c.blockDetails.AsData()
	}
	if c.execDetails != nil {
		data[ExecDetailsDataKey] = c.execDetails.AsData()
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

	var id uuid.UUID
	var err error

	if idData, found := data[ContentIDDataKey]; found {
		if idStr, ok := idData.(plugindata.String); ok {
			id, err = uuid.Parse(string(idStr))
			if err != nil {
				return nil, fmt.Errorf("error while parsing ID value: %w", err)
			}
		} else {
			return nil, fmt.Errorf("invalid ID value type found: %T", idData)
		}
	}

	blockDetailsMap, ok := data[BlockDetailsDataKey].(plugindata.Map)
	if !ok {
		return nil, errors.New("no block details found")
	}
	blockDetails, err := ParseBlockDetails(blockDetailsMap)
	if err != nil {
		return nil, err
	}

	execDetailsMap, ok := data[ExecDetailsDataKey].(plugindata.Map)
	if !ok {
		return nil, errors.New("no exec details found")
	}
	execDetails, err := ParseExecDetails(execDetailsMap)
	if err != nil {
		return nil, err
	}

	meta, ok := data[definitions.BlockKindMeta].(plugindata.Map)
	if !ok {
		return nil, errors.New("no meta found")
	}

	attrs := data[AttrsDataKey].(plugindata.Map)

	el := NewContentElement(
		id,
		ContentKind(kind),
		blockDetails,
		execDetails,
		meta,
		attrs,
	)
	return el, nil
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
	default:
		return parseContentElement(contentKind, data)
	}
}

func parseContentSection(data plugindata.Map) (*ContentSection, error) {
	if data == nil {
		return nil, nil
	}

	var id uuid.UUID
	var err error

	if idData, found := data[ContentIDDataKey]; found {
		if idStr, ok := idData.(plugindata.String); ok {
			id, err = uuid.Parse(string(idStr))
			if err != nil {
				return nil, fmt.Errorf("error while parsing ID value: %w", err)
			}
		} else {
			return nil, fmt.Errorf("invalid ID value type found: %T", idData)
		}
	}

	var title Content
	titleVal, ok := data["title"]
	if ok {
		titleValMap := titleVal.(plugindata.Map)
		var err error
		title, err = ParseContentData(titleValMap)
		if err != nil {
			return nil, err
		}
	}

	childrenVals, ok := data["children"].(plugindata.List)
	if !ok {
		return nil, fmt.Errorf("missing content section children")
	}

	children := make([]Content, len(childrenVals))
	for i, child := range childrenVals {
		children[i], err = ParseContentData(child.(plugindata.Map))
		if err != nil {
			return nil, err
		}
	}

	meta, ok := data[definitions.BlockKindMeta].(plugindata.Map)
	if !ok {
		return nil, errors.New("Meta not found in a section data")
	}

	detailsVal, ok := data[BlockDetailsDataKey].(plugindata.Map)
	if !ok {
		return nil, errors.New("block details not found in a section data")
	}

	details, err := ParseBlockDetails(detailsVal)
	if err != nil {
		return nil, err
	}

	section := NewSection(id, details, meta, title, children)
	return section, nil
}

func newContentID() uuid.UUID {
	uid, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Sprintf("Error generating a UUID v7: %v", err))
	}
	return uid
}
