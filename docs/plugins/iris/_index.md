---
title: blackstork/iris
weight: 20
plugin:
  name: blackstork/iris
  description: "The `iris` plugin for Iris Incident Response platform."
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/iris/"
type: docs
hideInMenu: true
---

{{< plugin-header "blackstork/iris" "iris" "v1.0.0" >}}

## Description
The `iris` plugin for Iris Incident Response platform.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use this plugin locally via `blackstork-cli`, you must declare it as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/iris" = ">= v1.0.0"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.


## Data sources

{{< plugin-resources "iris" "data-source" >}}
