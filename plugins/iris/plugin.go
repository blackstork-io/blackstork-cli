package iris

import (
	"errors"
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/fabric/internal/iris/client"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
)

type ClientLoadFn func(url, apiKey string, insecure bool) client.Client

var DefaultClientLoader ClientLoadFn = client.New

func Plugin(version string, loader ClientLoadFn) *plugin.Schema {
	if loader == nil {
		loader = DefaultClientLoader
	}
	return &plugin.Schema{
		Name:    "blackstork/iris",
		Doc:     "The `iris` plugin for Iris Incident Response platform.",
		Version: version,
		DataSources: plugin.DataSources{
			"iris_cases":  makeIrisCasesDataSource(loader),
			"iris_alerts": makeIrisAlertsDataSource(loader),
		},
	}
}

func parseConfig(cfg *dataspec.Block, loader ClientLoadFn) (client.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}
	apiURL := cfg.GetAttrVal("api_url").AsString()
	apiKey := cfg.GetAttrVal("api_key").AsString()
	insecure := cfg.GetAttrVal("insecure").True()
	return loader(apiURL, apiKey, insecure), nil
}

func handleClientError(err error) diagnostics.Diag {
	var clientErr *client.Error
	if errors.As(err, &clientErr) {
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to call Iris API",
			Detail:   clientErr.Message,
		}}
	}
	return diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Unknown error while calling Iris API",
	}}
}
