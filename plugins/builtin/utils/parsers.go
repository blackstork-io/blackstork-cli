// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

// Package utils provides helpers shared by built-in plugin components.
package utils

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"io"

	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
)

func ParseCSVContent(ctx context.Context, reader *csv.Reader) (plugindata.List, error) {
	rowMaps := make(plugindata.List, 0)
	headers, err := reader.Read()
	if err == io.EOF {
		return rowMaps, nil
	} else if err != nil {
		return nil, err
	}

	for {
		select {
		case <-ctx.Done(): // stop reading if the context is canceled
			return nil, ctx.Err()
		default:
			row, err := reader.Read()
			if err == io.EOF {
				return rowMaps, nil
			} else if err != nil {
				return nil, err
			}
			rowMap := make(plugindata.Map, len(headers))
			for j, header := range headers {
				if header == "" {
					continue
				}
				if j >= len(row) {
					rowMap[header] = nil
					continue
				}

				val := row[j]
				switch val {
				case "true":
					rowMap[header] = plugindata.Bool(true)
				case "false":
					rowMap[header] = plugindata.Bool(false)
				default:
					n := json.Number(val)
					if f, err := n.Float64(); err == nil {
						rowMap[header] = plugindata.Number(f)
					} else {
						rowMap[header] = plugindata.String(val)
					}
				}
			}
			rowMaps = append(rowMaps, rowMap)
		}
	}
}
