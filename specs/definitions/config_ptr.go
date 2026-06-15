// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package definitions

import (
	"context"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/blackstork-cli/parser/evaluation"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

// ConfigPtr is an attribute referencing a configuration block (`config = path.to.config`).
type ConfigPtr struct {
	Cfg *ConfigDef
	Ptr *hcl.Attribute
}

// Exists implements evaluation.Configuration.
func (c *ConfigPtr) Exists() bool {
	return c != nil
}

// ParseConfig implements Configuration.
func (c *ConfigPtr) ParseConfig(ctx context.Context, spec *dataspec.RootSpec) (val *dataspec.Block, diags diagnostics.Diag) {
	return c.Cfg.ParseConfig(ctx, spec)
}

// Range implements Configuration.
func (c *ConfigPtr) Range() hcl.Range {
	// Use the location of "config = *traversal*" for error reporting, not original config's Range
	return c.Ptr.Range
}

var _ evaluation.Configuration = (*ConfigPtr)(nil)
