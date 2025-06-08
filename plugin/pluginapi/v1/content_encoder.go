package pluginapiv1

import (
	"fmt"
	"log/slog"

	"github.com/blackstork-io/fabric/pkg/utils"
	"github.com/blackstork-io/fabric/plugin"
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
		el := &ContentElement{
			Meta:        encodeMetadata(val.Meta()),
			Kind:        string(val.Kind()),
			Attrs:       encodeMapData(val.Attrs()),
			//DataContext: encodeMapData(val.DataContext()),
		}
		variant = &Content_Element{
			Element: el,
		}
	case *plugin.ContentSection:
		if val == nil {
			break
		}
		variant = &Content_Section{
			Section: &ContentSection{
				Children: utils.FnMap(val.Children, EncodeContent),
				Meta:     encodeMetadata(val.Meta()),
			},
		}
	case *plugin.ContentEmpty:
		variant = &Content_Empty{
			Empty: &ContentEmpty{
				Meta: encodeMetadata(val.Meta()),
			},
		}
	default:
		slog.Error("Unknown content type encountered during encoding", "type", fmt.Sprintf("%T", src))
	}

	return &Content{
		Value: variant,
	}
}

func encodeMetadata(src *plugin.ContentMeta) *Metadata {
	if src == nil {
		return nil
	}
	return &Metadata{
		ProviderName:          src.ProviderName,
		ProviderPluginName:    src.ProviderPluginName,
		ProviderPluginVersion: src.ProviderPluginVersion,
	}
}
