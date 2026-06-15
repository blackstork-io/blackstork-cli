---
title: "`postgresql` data source"
plugin:
  name: blackstork/postgresql
  description: ""
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/postgresql/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/postgresql" "postgresql" "v0.4.2" "postgresql" "data source" >}}

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `postgresql` data source locally via `blackstork-cli`, you must declare the `blackstork/postgresql` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/postgresql" = ">= v0.4.2"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data postgresql` block:

```hcl
config data postgresql {
  # Required string.
  # Must be non-empty
  #
  # For example:
  database_url = "some string"
}
```

## Usage

This data source accepts the following arguments within a `data postgresql` block:

```hcl
data postgresql {
  # Required string.
  # Must be non-empty
  #
  # For example:
  sql_query = "some string"

  # Optional list of any single type.
  # Default value:
  sql_args = null
}
```