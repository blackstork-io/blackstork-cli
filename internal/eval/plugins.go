package eval

import (
	"github.com/blackstork-io/fabric/internal/plugin"
)

type DataSources interface {
	DataSource(name string) (*plugin.DataSource, bool)
}

type ContentProviders interface {
	ContentProvider(name string) (*plugin.ContentProvider, bool)
}

type Publishers interface {
	Publisher(name string) (*plugin.Publisher, bool)
}

type Formatters interface {
	Formatter(name string) (*plugin.Formatter, bool)
}

type Runners interface {
	DataSources
	ContentProviders
	Publishers
	Formatters
}
