package virustotal

import (
	"github.com/blackstork-io/fabric/internal/virustotal/client"
	"github.com/blackstork-io/fabric/internal/plugin"
)

type ClientLoadFn func(key string) client.Client

var DefaultClientLoader ClientLoadFn = client.New

func Plugin(version string, loader ClientLoadFn) *plugin.Schema {
	if loader == nil {
		loader = DefaultClientLoader
	}
	return &plugin.Schema{
		Name:    "blackstork/virustotal",
		Version: version,
		DataSources: plugin.DataSources{
			"virustotal_api_usage": makeVirusTotalAPIUsageDataSchema(loader),
		},
	}
}
