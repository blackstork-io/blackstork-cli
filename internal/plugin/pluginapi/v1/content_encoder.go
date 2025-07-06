package pluginapiv1

import (
	"fmt"
	"log/slog"

	"github.com/blackstork-io/fabric/internal/utils"
	"github.com/blackstork-io/fabric/internal/plugin"
)

func encodeContentProviderResult(src *plugin.ContentProviderResult) *ContentProviderResult {
	if src == nil {
		return nil
	}
	return &ContentProviderResult{
		Content: EncodeContent(src.Content),
	}
}

func EncodeContent(src plugin.Content) *Content {
	var variant isContent_Value
	switch val := src.(type) {
	case nil:
		return nil
	case *plugin.ContentElement:
		if val == nil {
			break
		}

		attrs := encodeMapData(val.Attrs())

		self := encodeBlockSelf(val.Self())
		meta := encodeMapData(val.Meta())

		el := &ContentElement{
			Self:  &self,
			Meta:  &meta,
			Kind:  string(val.Kind()),
			Attrs: &attrs,
			//DataContext: encodeMapData(val.DataContext()),
		}
		variant = &Content_Element{
			Element: el,
		}
	case *plugin.ContentSection:
		if val == nil {
			break
		}

		self := encodeBlockSelf(val.Self())
		meta := encodeMapData(val.Meta())

		variant = &Content_Section{
			Section: &ContentSection{
				Self:     &self,
				Children: utils.FnMap(val.Children, EncodeContent),
				Meta:     &meta,
			},
		}
	case *plugin.ContentEmpty:

		self := encodeBlockSelf(val.Self())
		meta := encodeMapData(val.Meta())

		variant = &Content_Empty{
			Empty: &ContentEmpty{
				Self: &self,
				Meta: &meta,
			},
		}
	default:
		slog.Error("Unknown content type encountered during encoding", "type", fmt.Sprintf("%T", src))
	}

	return &Content{
		Value: variant,
	}
}

func encodeBlockSelf(src plugin.BlockSelf) BlockSelf {
	return BlockSelf{
		Name:          src.Name,
		PluginName:    src.PluginName,
		PluginVersion: src.PluginVersion,
		ProviderName:  src.ProviderName,
	}
}
