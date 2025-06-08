package eval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/utils"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

const (
	maxDataSourceWorkers = 15
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

		if dataSourceName != "" && block.BlockRunnerName != dataSourceName {
			continue
		}

		if blockName != "" && block.BlockName != blockName {
			continue
		}

		matching = append(matching, block)
	}
	return matching
}


func executeDataBlocksAsync(ctx context.Context, doc *Document, path []string) (result plugindata.Data, diags diagnostics.Diag) {

	log := fabctx.GetLog(ctx)
	log = log.With("path", path)

	blocks := doc.DataBlocks

	log.InfoContext(
		ctx, "Executing the data blocks",
		"blocks_count", len(blocks),
	)

	if path != nil {
		blocks = filterBlocksByPath(blocks, path)
	}

	if len(blocks) == 0 {
		log.DebugContext(ctx, "No data blocks to execute")
		return result, diags
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
				"plugin_name", block.BlockRunnerName,
				"block_name", block.BlockName,
			)
			data, diags := block.FetchData(ctx)
			return &dataResult{
				pluginName: block.BlockRunnerName,
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
