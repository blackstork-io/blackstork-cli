package definitions

import (
	"context"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/fabric/parser/evaluation"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin/dataspec"
)

// Attribute referencing a configuration block (`config = path.to.config`).
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
