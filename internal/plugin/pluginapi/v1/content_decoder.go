package pluginapiv1

import (
	"fmt"
	"log/slog"

	"github.com/blackstork-io/fabric/internal/utils"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

func decodeContentProviderResult(src *ContentProviderResult) (*plugin.ContentProviderResult, error) {
	if src == nil {
		return nil, nil
	}

	content, err := decodeContent(src.Content)
	if err != nil {
		return nil, err
	}

	return &plugin.ContentProviderResult{Content: content}, nil
}

func decodeContent(src *Content) (plugin.Content, error) {
	switch val := src.GetValue().(type) {
	case *Content_Element:
		kind := plugin.ContentKind(val.Element.GetKind())

		self := decodeBlockSelf(val.Element.GetSelf())
		var meta plugindata.Map
		if val.Element.GetMeta() != nil {
			meta = decodeMapData(val.Element.GetMeta().GetValue())
		}
		attrs := decodeMapData(val.Element.GetAttrs().GetValue())

		el, err := plugin.NewContentElement(kind, self, meta, attrs)
		if err != nil {
			return nil, err
		}
		return el, nil
	case *Content_Section:

		self := decodeBlockSelf(val.Section.GetSelf())
		var meta plugindata.Map
		if val.Section.GetMeta() != nil {
			meta = decodeMapData(val.Section.GetMeta().GetValue())
		}

		children, err := utils.FnMapErr(
			val.Section.GetChildren(),
			decodeContent,
		)
		if err != nil {
			return nil, err
		}

		section := plugin.NewSection(self, meta, children)
		return section, nil
	case *Content_Empty:

		self := decodeBlockSelf(val.Empty.GetSelf())
		var meta plugindata.Map
		if val.Empty.GetMeta() != nil {
			meta = decodeMapData(val.Empty.GetMeta().GetValue())
		}

		return plugin.NewEmptyContent(self, meta), nil
	case nil:
		slog.Error("Received nil content", "src", src)
		return nil, nil
	default:
		slog.Error("Unknown content type encountered during decoding", "type", fmt.Sprintf("%T", src))
		return nil, fmt.Errorf("unknown content type: %T", src)
	}
}

func decodeBlockSelf(src *BlockSelf) plugin.BlockSelf {
	return plugin.BlockSelf{
		Name:          src.Name,
		PluginName:    src.PluginName,
		PluginVersion: src.PluginVersion,
		ProviderName:  src.ProviderName,
	}
}

func decodeFormattedContent(src *FormattedContent) *plugin.FormattedContent {
	meta := decodeMapData(src.GetMeta().GetValue())

	return &plugin.FormattedContent{
		Self:    decodeBlockSelf(src.Self),
		Meta:    meta,
		Content: src.Content,
		Format:  src.Format,
	}
}
