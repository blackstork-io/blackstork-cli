package utils

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"io"

	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
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
