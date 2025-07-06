package openai

import (
	"github.com/blackstork-io/fabric/internal/openai/client"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
)

const (
	defaultModel = "gpt-3.5-turbo"
)

type ClientLoadFn func(opts ...client.Option) client.Client

var DefaultClientLoader ClientLoadFn = client.New

func Plugin(version string, loader ClientLoadFn) *plugin.Schema {
	if loader == nil {
		loader = DefaultClientLoader
	}
	return &plugin.Schema{
		Name:    "blackstork/openai",
		Version: version,
		ContentProviders: plugin.ContentProviders{
			"openai_text": makeOpenAITextContentSchema(loader),
		},
	}
}

func makeClient(loader ClientLoadFn, cfg *dataspec.Block) (client.Client, error) {
	opts := []client.Option{
		client.WithAPIKey(cfg.GetAttrVal("api_key").AsString()),
	}
	orgID := cfg.GetAttrVal("organization_id")
	if !orgID.IsNull() && orgID.AsString() != "" {
		opts = append(opts, client.WithOrgID(orgID.AsString()))
	}
	return loader(opts...), nil
}
