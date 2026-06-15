---
title: "`opencti` data source"
plugin:
  name: blackstork/opencti
  description: ""
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/opencti/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/opencti" "opencti" "v0.4.2" "opencti" "data source" >}}

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `opencti` data source locally via `blackstork-cli`, you must declare the `blackstork/opencti` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/opencti" = ">= v0.4.2"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data opencti` block:

```hcl
config data opencti {
  # Required string.
  #
  # For example:
  graphql_url = "some string"

  # Optional string.
  # Default value:
  auth_token = null
}
```

## Usage

This data source accepts the following arguments within a `data opencti` block:

```hcl
data opencti {
  # Required string.
  #
  # For example:
  graphql_query = "some string"
}
```