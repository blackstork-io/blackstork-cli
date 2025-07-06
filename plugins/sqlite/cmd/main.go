package main

import (
	"github.com/blackstork-io/fabric/internal/sqlite"
	pluginapiv1 "github.com/blackstork-io/fabric/internal/plugin/pluginapi/v1"
)

var version string

func main() {
	pluginapiv1.Serve(
		sqlite.Plugin(version),
	)
}
