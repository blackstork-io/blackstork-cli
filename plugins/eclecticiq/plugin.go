// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package eclecticiq

import (
	"log/slog"

	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugins/eclecticiq/client"
)

type EIQClientLoaderFn func(url, apiKey string) (client.Client, error)

var DefaultEIQClientLoader EIQClientLoaderFn = client.New

func Plugin(version string, loader EIQClientLoaderFn) *plugin.Schema {
	log := slog.Default()
	log = log.With("data_source", "eclecticiq_entities")

	if loader == nil {
		loader = DefaultEIQClientLoader
	}
	return &plugin.Schema{
		Name:    "blackstork/eclecticiq",
		Version: version,
		DataSources: plugin.DataSources{
			"eclecticiq_entities": makeEIQEntitiesDataSource(log, loader),
		},
	}
}
