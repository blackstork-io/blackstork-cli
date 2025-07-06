package hackerone

import (
	"fmt"

	"github.com/blackstork-io/fabric/internal/hackerone/client"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
)

const (
	minPage  = 1
	pageSize = 25
)

type ClientLoadFn func(user, token string) client.Client

var DefaultClientLoader ClientLoadFn = client.New

func Plugin(version string, loader ClientLoadFn) *plugin.Schema {
	if loader == nil {
		loader = DefaultClientLoader
	}
	return &plugin.Schema{
		Name:    "blackstork/hackerone",
		Version: version,
		DataSources: plugin.DataSources{
			"hackerone_reports": makeHackerOneReportsDataSchema(loader),
		},
	}
}

func makeClient(loader ClientLoadFn, cfg *dataspec.Block) (client.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}
	user := cfg.GetAttrVal("api_username")
	if user.IsNull() || user.AsString() == "" {
		return nil, fmt.Errorf("api_username is required in configuration")
	}
	token := cfg.GetAttrVal("api_token")
	if token.IsNull() || token.AsString() == "" {
		return nil, fmt.Errorf("api_token is required in configuration")
	}
	return loader(user.AsString(), token.AsString()), nil
}
