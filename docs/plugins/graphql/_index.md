---
title: blackstork/graphql
weight: 20
plugin:
  name: blackstork/graphql
  description: ""
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/graphql/"
type: docs
hideInMenu: true
---

{{< plugin-header "blackstork/graphql" "graphql" "v1.0.0" >}}

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use this plugin locally via `blackstork-cli`, you must declare it as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/graphql" = ">= v1.0.0"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.


## Data sources

{{< plugin-resources "graphql" "data-source" >}}
