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

	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
)

func DecodeData(src *Data) plugindata.Data {
	switch src.GetData().(type) {
	case nil:
		return nil
	case *Data_NumberVal:
		return plugindata.Number(src.GetNumberVal())
	case *Data_StringVal:
		return plugindata.String(src.GetStringVal())
	case *Data_BoolVal:
		return plugindata.Bool(src.GetBoolVal())
	case *Data_MapVal:
		return decodeMapData(src.GetMapVal().GetValue())
	case *Data_ListVal:
		return plugindata.List(utils.FnMap(src.GetListVal().GetValue(), DecodeData))
	case *Data_TimeVal:
		return plugindata.Time(src.GetTimeVal().AsTime())
	}
	panic(fmt.Sprintf("Unexpected src data type: %T", src.GetData()))
}

func decodeMapData(src map[string]*Data) plugindata.Map {
	return plugindata.Map(utils.MapMap(src, DecodeData))
}
