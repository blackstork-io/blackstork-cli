package basic

import "github.com/blackstork-io/fabric/internal/plugin"

// Plugin returns the schema and version for the plugin
// This is the entry point for the plugin
func Plugin(version string) *plugin.Schema {
	return &plugin.Schema{
		// Name is the name of the plugin
		Name: "example/basic",
		// Version is the version of the plugin
		// It must be a valid semver
		Version: version,
		// DataSources is a map of data sources that the plugin provides
		// The key is the name of the data source and the value is the data source schema
		DataSources: plugin.DataSources{
			"basic_random_numbers": makeRandomNumbersDataSource(),
		},
		// ContentProviders is a map of content providers that the plugin provides
		// The key is the name of the content provider and the value is the content provider schema
		ContentProviders: plugin.ContentProviders{
			"basic_greeting": makeGreetingContentProvider(),
		},
	}
}
