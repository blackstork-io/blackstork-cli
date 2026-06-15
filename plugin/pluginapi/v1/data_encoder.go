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
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
)

func EncodeData(d plugindata.Data) *Data {
	switch v := d.(type) {
	case nil:
		return nil
	case plugindata.Number:
		return &Data{
			Data: &Data_NumberVal{
				NumberVal: float64(v),
			},
		}
	case plugindata.String:
		return &Data{
			Data: &Data_StringVal{
				StringVal: string(v),
			},
		}
	case plugindata.Bool:
		return &Data{
			Data: &Data_BoolVal{
				BoolVal: bool(v),
			},
		}
	case plugindata.Map:
		mapData := encodeMapData(v)
		return &Data{
			Data: &Data_MapVal{
				&mapData,
			},
		}
	case plugindata.List:
		return &Data{
			Data: &Data_ListVal{
				ListVal: &ListData{
					Value: utils.FnMap(v, EncodeData),
				},
			},
		}
	case plugindata.Time:
		return &Data{
			Data: &Data_TimeVal{
				TimeVal: timestamppb.New(time.Time(d.(plugindata.Time))),
			},
		}
	default:
		if cd, ok := d.(plugindata.Convertible); ok {
			return EncodeData(cd.AsPluginData())
		}
	}
	panic(fmt.Errorf("unexpected plugin data type: %T", d))
}

func encodeMapData(m plugindata.Map) MapData {
	return MapData{
		Value: utils.MapMap(m, EncodeData),
	}
}
