// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package eval

import (
	"strings"

	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

type PluginAction struct {
	Source definitions.ExecBlock

	RunnerName string
	BlockName  string

	meta   *definitions.MetaBlock
	Config *dataspec.Block
	Args   *dataspec.Block
}

func (a *PluginAction) FullBlockName() string {
	blocks := []string{}

	// Source can be nil for manually created actions
	if a.Source != nil {
		// block kind
		blocks = append(blocks, a.Source.GetSourceKind())
	}

	blocks = append(blocks, a.RunnerName)

	if a.BlockName != "" {
		blocks = append(blocks, a.BlockName)
	}

	return strings.Join(blocks, ".")
}
