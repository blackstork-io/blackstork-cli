---
title: "`sqlite` data source"
plugin:
  name: blackstork/sqlite
  description: ""
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/sqlite/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/sqlite" "sqlite" "v0.4.2" "sqlite" "data source" >}}

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `sqlite` data source locally via `blackstork-cli`, you must declare the `blackstork/sqlite` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/sqlite" = ">= v0.4.2"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data sqlite` block:

```hcl
config data sqlite {
  # Required string.
  # Must be non-empty
  #
  # For example:
  database_uri = "some string"
}
```

## Usage

This data source accepts the following arguments within a `data sqlite` block:

```hcl
data sqlite {
  # SQL query to execute
  #
  # Required string.
  #
  # For example:
  sql_query = "some string"

  # A tuple (or list) of strings, numbers, or booleans to be used as arguments in the SQL query
  #
  # Optional any type.
  #
  # For example:
  # sql_args = ["example argument", 2, false]
  #
  # Default value:
  sql_args = null
}
```