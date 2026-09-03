---
title: "`graphql` data source"
plugin:
  name: blackstork/graphql
  description: "Executes a GraphQL query against an HTTP endpoint and returns the decoded JSON response. Supports bearer-token authentication"
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/graphql/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/graphql" "graphql" "v1.0.0" "graphql" "data source" >}}

## Description
Executes a GraphQL query against an HTTP endpoint and returns the decoded JSON response. Supports bearer-token authentication.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `graphql` data source locally via `blackstork-cli`, you must declare the `blackstork/graphql` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/graphql" = ">= v1.0.0"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data graphql` block:

```hcl
config data graphql {
  # Required string.
  #
  # For example:
  url = "some string"

  # Optional string.
  # Default value:
  auth_token = null
}
```

## Usage

This data source accepts the following arguments within a `data graphql` block:

```hcl
data graphql {
  # Required string.
  #
  # For example:
  query = "some string"
}
```