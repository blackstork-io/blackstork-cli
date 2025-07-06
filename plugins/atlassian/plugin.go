package atlassian

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/fabric/internal/atlassian/client"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
)

type ClientLoadFn func(url, accountEmail, apiToken string) client.Client

var DefaultClientLoader ClientLoadFn = client.New

func Plugin(version string, loader ClientLoadFn) *plugin.Schema {
	if loader == nil {
		loader = DefaultClientLoader
	}
	return &plugin.Schema{
		Name:    "blackstork/atlassian",
		Doc:     "The `atlassian` plugin for Atlassian Cloud.",
		Version: version,
		DataSources: plugin.DataSources{
			"jira_issues": makeJiraIssuesDataSource(loader),
		},
	}
}

func parseConfig(cfg *dataspec.Block, loader ClientLoadFn) (client.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}
	domain := cfg.GetAttrVal("domain").AsString()
	apiURL := fmt.Sprintf("https://%s.atlassian.net", domain)
	accountEmail := cfg.GetAttrVal("account_email").AsString()
	apiToken := cfg.GetAttrVal("api_token").AsString()
	return loader(apiURL, accountEmail, apiToken), nil
}

func handleClientError(err error) diagnostics.Diag {
	var clientErr *client.Error
	if errors.As(err, &clientErr) {
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Failed to call Atlassian API",
			Detail:   strings.Join(clientErr.ErrorMessages, " "),
		}}
	}
	return diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Unknown error while calling Atlassian API",
	}}
}
