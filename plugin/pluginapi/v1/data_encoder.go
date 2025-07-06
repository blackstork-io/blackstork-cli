package pluginapiv1

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/blackstork-io/fabric/pkg/utils"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

func encodeData(d plugindata.Data) *Data {
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
					Value: utils.FnMap(v, encodeData),
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
			return encodeData(cd.AsPluginData())
		}
	}
	panic(fmt.Errorf("unexpected plugin data type: %T", d))
}

func encodeMapData(m plugindata.Map) MapData {
	return MapData{
		Value: utils.MapMap(m, encodeData),
	}
}
