// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package eval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
)

const (
	maxDataSourceWorkers = 50
)

type dataResult struct {
	pluginName string
	blockName  string
	data       plugindata.Data
	diags      diagnostics.Diag
}

func filterBlocksByPath(blocks []*PluginDataAction, path []string) (matching []*PluginDataAction) {
	var dataSourceName string
	if len(path) > 0 {
		dataSourceName = path[0]
	}

	var blockName string
	if len(path) > 1 {
		blockName = path[1]
	}

	matching = []*PluginDataAction{}
	for i := range blocks {
		block := blocks[i]

		if dataSourceName != "" && block.RunnerName != dataSourceName {
			continue
		}

		if blockName != "" && block.BlockName != blockName {
			continue
		}

		matching = append(matching, block)
	}
	return matching
}

func executeDataBlocksAsync(ctx context.Context, doc *Document, path []string) (_ plugindata.Map, diags diagnostics.Diag) {
	log := appctx.Log(ctx)
	log = log.With("path", path)

	blocks := doc.DataBlocks

	if len(blocks) == 0 {
		log.DebugContext(ctx, "No data blocks found")
		return nil, diags
	}

	log.InfoContext(
		ctx, "Executing data blocks",
		"blocks_count", len(blocks),
	)

	if path != nil {
		blocks = filterBlocksByPath(blocks, path)
	}

	if len(blocks) == 0 {
		log.DebugContext(ctx, "No data blocks found")
		return plugindata.Map{}, diags
	}

	workersCount := min(len(blocks), maxDataSourceWorkers)

	results, _ := utils.RunWorkers(
		ctx,
		log.With("task", "execute_data_blocks"),
		blocks,
		workersCount,
		func(block *PluginDataAction) (*dataResult, error) {
			log.DebugContext(
				ctx, "Executing a data block",
				"plugin_name", block.RunnerName,
				"block_name", block.BlockName,
			)
			data, diags := block.FetchData(ctx)
			return &dataResult{
				pluginName: block.RunnerName,
				blockName:  block.BlockName,
				data:       data,
				diags:      diags,
			}, nil
		},
	)

	for _, res := range results {
		for _, diag := range res.diags {
			diags.Append(diag)
		}
	}
	// Return with all collected errors
	if diags.HasErrors() {
		return nil, diags
	}

	data := make(plugindata.Map)

	for _, res := range results {
		var dataSourceMap plugindata.Map

		if found, ok := data[res.pluginName]; ok {
			dataSourceMap, ok = found.(plugindata.Map)
			if !ok {
				log.ErrorContext(
					ctx, "Expected a map from a data source but received something else",
					"plugin_name", res.pluginName,
					"block_name", res.blockName,
					"unexpected_type", fmt.Sprintf("%T", found),
				)
				return nil, diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Unexpected data type",
					Detail:   fmt.Sprintf("Unexpected data type returned by data source `%s`", res.pluginName),
				}}
			}
		} else {
			dataSourceMap = make(plugindata.Map)
			data[res.pluginName] = dataSourceMap
		}

		dataSourceMap[res.blockName] = res.data
	}

	return data, diags
}
