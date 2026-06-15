// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package nistnvd

import (
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugins/nistnvd/client"
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
