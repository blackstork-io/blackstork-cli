package main

import (
	"github.com/blackstork-io/fabric/internal/iris"
	pluginapiv1 "github.com/blackstork-io/fabric/internal/plugin/pluginapi/v1"
)

var version string

func main() {
	pluginapiv1.Serve(
		iris.Plugin(version, iris.DefaultClientLoader),
	)
}
