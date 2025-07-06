package nistnvd

import (
	"github.com/blackstork-io/fabric/internal/nistnvd/client"
	"github.com/blackstork-io/fabric/internal/plugin"
)

type ClientLoadFn func(apiKey *string) client.Client

var DefaultClientLoader ClientLoadFn = client.New

func Plugin(version string, loader ClientLoadFn) *plugin.Schema {
	return &plugin.Schema{
		Name:    "blackstork/nist_nvd",
		Version: version,
		DataSources: plugin.DataSources{
			"nist_nvd_cves": makeNistNvdCvesDataSource(loader),
		},
	}
}
