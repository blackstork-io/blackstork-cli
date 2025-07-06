package sqlite

import (
	"github.com/blackstork-io/fabric/internal/plugin"
)

func Plugin(version string) *plugin.Schema {
	return &plugin.Schema{
		Name:    "blackstork/sqlite",
		Version: version,
		DataSources: plugin.DataSources{
			"sqlite": makeSqliteDataSource(),
		},
	}
}
