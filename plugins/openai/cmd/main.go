package main

import (
	"github.com/blackstork-io/fabric/internal/openai"
	pluginapiv1 "github.com/blackstork-io/fabric/internal/plugin/pluginapi/v1"
)

var version string

func main() {
	pluginapiv1.Serve(
		openai.Plugin(version, nil),
	)
}
