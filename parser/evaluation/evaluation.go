// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package evaluation

import (
	"context"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

// To act as a plugin configuration struct must implement this interface.
type Configuration interface {
	ParseConfig(ctx context.Context, spec *dataspec.RootSpec) (*dataspec.Block, diagnostics.Diag)
	Range() hcl.Range
}
