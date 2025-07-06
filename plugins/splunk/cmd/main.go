package main

import (
	"github.com/blackstork-io/fabric/internal/splunk"
	pluginapiv1 "github.com/blackstork-io/fabric/internal/plugin/pluginapi/v1"
)

var version string

func main() {
	pluginapiv1.Serve(
		splunk.Plugin(version, splunk.DefaultClientLoader),
	)
}
