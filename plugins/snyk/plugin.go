package snyk

import (
	"github.com/blackstork-io/fabric/internal/snyk/client"
	"github.com/blackstork-io/fabric/internal/plugin"
)

const (
	pageSize = 100
)

type ClientLoadFn func(apiKey string) client.Client

var DefaultClientLoader ClientLoadFn = client.New

func Plugin(version string, loader ClientLoadFn) *plugin.Schema {
	if loader == nil {
		loader = DefaultClientLoader
	}
	return &plugin.Schema{
		Name:    "blackstork/snyk",
		Version: version,
		DataSources: plugin.DataSources{
			"snyk_issues": makeSnykIssuesDataSource(loader),
		},
	}
}
