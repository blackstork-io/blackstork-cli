package main

import (
	"github.com/blackstork-io/fabric/internal/opencti"
	pluginapiv1 "github.com/blackstork-io/fabric/internal/plugin/pluginapi/v1"
)

var version string

func main() {
	pluginapiv1.Serve(
		opencti.Plugin(version),
	)
}
