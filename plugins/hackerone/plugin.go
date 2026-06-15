// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package hackerone

import (
	"fmt"

	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugins/hackerone/client"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
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
