package main

import (
	"github.com/blackstork-io/fabric/internal/elastic"
	pluginapiv1 "github.com/blackstork-io/fabric/internal/plugin/pluginapi/v1"
)

var version string

func main() {
	pluginapiv1.Serve(
		elastic.Plugin(version, elastic.DefaultKibanaClientLoader),
	)
}
