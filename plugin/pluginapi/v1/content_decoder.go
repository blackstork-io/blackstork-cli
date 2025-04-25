package pluginapiv1

import (
	"fmt"
	"log/slog"

	"github.com/blackstork-io/fabric/pkg/utils"
	"github.com/blackstork-io/fabric/plugin"
)

func decodeContentResult(src *ContentResult) (*plugin.ContentResult, error) {
	if src == nil {
		return nil, nil
	}

	content, err := decodeContent(src.Content)
	if err != nil {
		return nil, err
	}

	return &plugin.ContentResult{Content: content}, nil
}

func decodeContent(src *Content) (plugin.Content, error) {
	switch val := src.GetValue().(type) {
	case *Content_Element:
		id := plugin.ContentID(val.Element.GetId())
		kind := plugin.ContentKind(val.Element.GetKind())

		meta := decodeMetadata(val.Element.GetMeta())
		attrs := decodeMapData(val.Element.GetAttrs().GetValue())
		data := decodeMapData(val.Element.GetDataContext().GetValue())

		el, err := plugin.NewContentElement(id, kind, meta, attrs, data)
		if err != nil {
			return nil, err
		}
		return el, nil
	case *Content_Section:
		id := plugin.ContentID(val.Section.GetId())
		meta := decodeMetadata(val.Section.GetMeta())

		children, err := utils.FnMapErr(
			val.Section.GetChildren(),
			decodeContent,
		)
		if err != nil {
			return nil, err
		}

		section := plugin.NewSection(id, meta, children)
		return section, nil
	case *Content_Empty:
		id := plugin.ContentID(val.Empty.GetId())
		meta := decodeMetadata(val.Empty.GetMeta())

		return plugin.NewEmptyContent(&id, meta), nil
	case nil:
		slog.Error("Received nil content", "src", src)
		return nil, nil
	default:
		slog.Error("Unknown content type encountered during decoding", "type", fmt.Sprintf("%T", src))
		return nil, fmt.Errorf("unknown content type: %T", src)
	}
}

func decodeMetadata(src *Metadata) *plugin.ContentMeta {
	if src == nil {
		return nil
	}
	return &plugin.ContentMeta{
		ProviderName: src.ProviderName,
		ProviderPluginName: src.ProviderPluginName,
		ProviderPluginVersion: src.ProviderPluginVersion,
	}
}

func decodeFormattedContent(src *FormattedContent) *plugin.FormattedContent {
	return &plugin.FormattedContent{
		Content: src.Content,
		Format: src.Format,
	}
}
