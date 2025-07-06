package splunk

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/splunk/client"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
)

type ClientLoadFn func(token, host, deployment string) client.Client

var DefaultClientLoader ClientLoadFn = client.New

func Plugin(version string, loader ClientLoadFn) *plugin.Schema {
	if loader == nil {
		loader = DefaultClientLoader
	}
	return &plugin.Schema{
		Name:    "blackstork/splunk",
		Version: version,
		DataSources: plugin.DataSources{
			"splunk_search": makeSplunkSearchDataSchema(loader),
		},
	}
}

func makeClient(loader ClientLoadFn, cfg *dataspec.Block) (client.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	token := cfg.GetAttrVal("auth_token")
	if token.IsNull() || token.AsString() == "" {
		return nil, fmt.Errorf("auth_token is required in configuration")
	}
	host := cfg.GetAttrVal("host")
	if host.IsNull() {
		host = cty.StringVal("")
	}
	deployment := cfg.GetAttrVal("deployment_name")
	if deployment.IsNull() {
		deployment = cty.StringVal("")
	}
	if host.AsString() == "" && deployment.AsString() == "" {
		return nil, fmt.Errorf("host or deployment_name is required in configuration")
	}
	return loader(token.AsString(), host.AsString(), deployment.AsString()), nil
}
