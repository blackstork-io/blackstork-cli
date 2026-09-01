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
	"math"

	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
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
		blockDetails := encodeBlockDetails(val.BlockDetails())
		execDetails := encodeExecDetails(val.ExecDetails())
		meta := encodeMapData(val.Meta())

		id := val.ID()

		variant = &Content_Element{
			Element: &ContentElement{
				Id:           id[:],
				BlockDetails: &blockDetails,
				ExecDetails:  &execDetails,
				Meta:         &meta,
				Kind:         string(val.Kind()),
				Attrs:        &attrs,
				// DataContext: encodeMapData(val.DataContext()),
			},
		}
	case *plugin.ContentSection:
		if val == nil {
			break
		}

		blockDetails := encodeBlockDetails(val.BlockDetails())
		meta := encodeMapData(val.Meta())

		id := val.ID()

		var title *Content
		if val.Title != nil {
			title = EncodeContent(val.Title)
		}

		variant = &Content_Section{
			Section: &ContentSection{
				Id:           id[:],
				BlockDetails: &blockDetails,
				Meta:         &meta,
				Title:        title,
				Children:     utils.FnMap(val.Children, EncodeContent),
			},
		}
	default:
		slog.Error("Unknown content type encountered during encoding", "type", fmt.Sprintf("%T", src))
	}

	return &Content{
		Value: variant,
	}
}

func encodeBlockDetails(src *plugin.BlockDetails) BlockDetails {
	depth := max(src.Depth, 0)
	if uint64(depth) > uint64(math.MaxUint32) {
		depth = math.MaxUint32
	}
	return BlockDetails{
		Kind:   src.Kind,
		Runner: src.Runner,
		Name:   src.Name,
		Id:     src.ID,
		Depth:  uint32(depth),
	}
}

func encodeExecDetails(src *plugin.ExecDetails) ExecDetails {
	return ExecDetails{
		PluginName:    src.PluginName,
		PluginVersion: src.PluginVersion,
		Runner:        src.Runner,
	}
}
