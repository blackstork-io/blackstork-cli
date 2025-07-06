package terraform

import (
	"github.com/blackstork-io/fabric/internal/plugin"
)

func Plugin(version string) *plugin.Schema {
	return &plugin.Schema{
		Name:    "blackstork/terraform",
		Version: version,
		DataSources: plugin.DataSources{
			"terraform_state_local": makeTerraformStateLocalDataSource(),
		},
	}
}
