// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package plugin

import (
	"errors"

	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
)

type ExecDetails struct {
	PluginName    string
	PluginVersion string
	Runner        string
}

func (s *ExecDetails) AsData() plugindata.Map {
	return plugindata.Map{
		"plugin_name":    plugindata.String(s.PluginName),
		"plugin_version": plugindata.String(s.PluginVersion),
		"runner":         plugindata.String(s.Runner),
	}
}

func ParseExecDetails(data plugindata.Data) (*ExecDetails, error) {
	if data == nil {
		return nil, errors.New("no block data found")
	}
	details := data.(plugindata.Map)

	plugin, _ := details["plugin_name"].(plugindata.String)
	version, _ := details["plugin_version"].(plugindata.String)
	runner, _ := details["runner"].(plugindata.String)

	return &ExecDetails{
		PluginName:    string(plugin),
		PluginVersion: string(version),
		Runner:        string(runner),
	}, nil
}
