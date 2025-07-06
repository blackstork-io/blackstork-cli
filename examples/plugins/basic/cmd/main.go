package main

import (
	"github.com/blackstork-io/fabric/examples/plugins/basic"
	pluginapiv1 "github.com/blackstork-io/fabric/internal/plugin/pluginapi/v1"
)

func main() {
	// Serve the plugin using the pluginapiv1
	pluginapiv1.Serve(
		// Pass the plugin schema to the Serve function
		basic.Plugin("0.0.1"),
	)
	// Thats it! The plugin is now ready to be used
	// Build it and put it in the plugin directory to test it out
	// Binary should be placed like this in the plugin directory
	//
	// 	/plugins
	//		/example
	// 			basic@0.0.1
}
