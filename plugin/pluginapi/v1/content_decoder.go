// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package pluginapiv1

import (
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
)

func decodeContentProviderResult(src *ContentProviderResult) (*plugin.ContentProviderResult, error) {
	if src == nil {
		return nil, nil
	}

	content, err := DecodeContent(src.Content)
	if err != nil {
		return nil, err
	}

	return &plugin.ContentProviderResult{Content: content}, nil
}

func DecodeContent(src *Content) (plugin.Content, error) {
	switch val := src.GetValue().(type) {
	case *Content_Element:
		kind := plugin.ContentKind(val.Element.GetKind())

		var id uuid.UUID

		if val.Element.Id != nil {
			id = uuid.UUID(val.Element.Id)
		}

		blockDetails := decodeBlockDetails(val.Element.BlockDetails)
		execDetails := decodeExecDetails(val.Element.ExecDetails)

		var meta plugindata.Map
		if val.Element.GetMeta() != nil {
			meta = decodeMapData(val.Element.GetMeta().GetValue())
		}
		attrs := decodeMapData(val.Element.GetAttrs().GetValue())

		el := plugin.NewContentElement(id, kind, blockDetails, execDetails, meta, attrs)
		return el, nil
	case *Content_Section:

		blockDetails := decodeBlockDetails(val.Section.BlockDetails)

		var meta plugindata.Map
		if val.Section.GetMeta() != nil {
			meta = decodeMapData(val.Section.GetMeta().GetValue())
		}

		children, err := utils.FnMapErr(
			val.Section.GetChildren(),
			DecodeContent,
		)
		if err != nil {
			return nil, err
		}

		var id uuid.UUID

		if val.Section.Id != nil {
			id = uuid.UUID(val.Section.Id)
		}

		titleVal := val.Section.GetTitle()
		var title plugin.Content
		if titleVal != nil {
			title, err = DecodeContent(titleVal)
			if err != nil {
				return nil, err
			}
		}

		section := plugin.NewSection(id, blockDetails, meta, title, children)
		return section, nil
	case nil:
		slog.Error("Received nil content", "src", src)
		return nil, nil
	default:
		slog.Error("Unknown content type encountered during decoding", "type", fmt.Sprintf("%T", src))
		return nil, fmt.Errorf("unknown content type: %T", src)
	}
}

func decodeBlockDetails(src *BlockDetails) *plugin.BlockDetails {
	return &plugin.BlockDetails{
		Kind:   src.Kind,
		Name:   src.Name,
		Runner: src.Runner,
		ID:     src.Id,
		Depth:  int(src.Depth),
	}
}

func decodeExecDetails(src *ExecDetails) *plugin.ExecDetails {
	return &plugin.ExecDetails{
		PluginName:    src.PluginName,
		PluginVersion: src.PluginVersion,
		Runner:        src.Runner,
	}
}

func decodeFormattedContent(src *FormattedContent) *plugin.FormattedContent {
	meta := decodeMapData(src.GetMeta().GetValue())

	return &plugin.FormattedContent{
		ExecDetails: decodeExecDetails(src.ExecDetails),
		Meta:        meta,
		Content:     src.Content,
		Format:      src.Format,
	}
}
