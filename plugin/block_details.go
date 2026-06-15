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

type BlockDetails struct {
	Kind   string
	Runner string
	Name   string

	ID    string
	Depth int
}

func (s BlockDetails) AsData() plugindata.Map {
	return plugindata.Map{
		"kind":   plugindata.String(s.Kind),
		"runner": plugindata.String(s.Runner),
		"name":   plugindata.String(s.Name),
		"id":     plugindata.String(s.ID),
		"depth":  plugindata.Number(s.Depth),
	}
}

func ParseBlockDetails(data plugindata.Data) (*BlockDetails, error) {
	if data == nil {
		return nil, errors.New("no block data found")
	}
	details, ok := data.(plugindata.Map)
	if !ok {
		return nil, errors.New("invalid type of block details data")
	}

	kind, _ := details["kind"].(plugindata.String)
	runner, _ := details["runner"].(plugindata.String)
	blockName, _ := details["name"].(plugindata.String)

	id, _ := details["id"].(plugindata.String)
	depth, _ := details["depth"].(plugindata.Number)

	return &BlockDetails{
		Kind:   string(kind),
		Runner: string(runner),
		Name:   string(blockName),
		ID:     string(id),
		Depth:  int(depth),
	}, nil
}
