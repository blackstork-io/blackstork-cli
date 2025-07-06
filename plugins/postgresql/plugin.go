package postgresql

import (
	"github.com/blackstork-io/fabric/internal/plugin"
)

func Plugin(version string) *plugin.Schema {
	return &plugin.Schema{
		Name:    "blackstork/postgresql",
		Version: version,
		DataSources: plugin.DataSources{
			"postgresql": makePostgreSQLDataSource(),
		},
	}
}
