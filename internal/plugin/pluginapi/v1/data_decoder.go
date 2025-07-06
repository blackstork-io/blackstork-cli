package pluginapiv1

import (
	"fmt"

	"github.com/blackstork-io/fabric/internal/utils"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

func decodeData(src *Data) plugindata.Data {
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
		return plugindata.List(utils.FnMap(src.GetListVal().GetValue(), decodeData))
	case *Data_TimeVal:
		return plugindata.Time(src.GetTimeVal().AsTime())
	}
	panic(fmt.Sprintf("Unexpected src data type: %T", src.GetData()))
}

func decodeMapData(src map[string]*Data) plugindata.Map {
	return plugindata.Map(utils.MapMap(src, decodeData))
}
