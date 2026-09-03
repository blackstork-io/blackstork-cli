---
title: "`postgresql` data source"
plugin:
  name: blackstork/postgresql
  description: "Runs a parameterized SQL query against PostgreSQL and returns the result rows as a list of objects keyed by column name"
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/postgresql/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/postgresql" "postgresql" "v1.0.0" "postgresql" "data source" >}}

## Description
Runs a parameterized SQL query against PostgreSQL and returns the result rows as a list of objects keyed by column name.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `postgresql` data source locally via `blackstork-cli`, you must declare the `blackstork/postgresql` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/postgresql" = ">= v1.0.0"
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