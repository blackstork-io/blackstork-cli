---
title: blackstork/postgresql
weight: 20
plugin:
  name: blackstork/postgresql
  description: ""
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/postgresql/"
type: docs
hideInMenu: true
---

{{< plugin-header "blackstork/postgresql" "postgresql" "v1.0.0-rc1" >}}

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use this plugin locally via `blackstork-cli`, you must declare it as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/postgresql" = ">= v1.0.0-rc1"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.


## Data sources

{{< plugin-resources "postgresql" "data-source" >}}
