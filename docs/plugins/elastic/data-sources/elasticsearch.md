---
title: "`elasticsearch` data source"
plugin:
  name: blackstork/elastic
  description: "Retrieves documents and aggregations from Elasticsearch by document ID, query string, or Query DSL query"
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/elastic/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/elastic" "elastic" "v1.0.0" "elasticsearch" "data source" >}}

## Description
Retrieves documents and aggregations from Elasticsearch by document ID, query string, or Query DSL query.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `elasticsearch` data source locally via `blackstork-cli`, you must declare the `blackstork/elastic` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/elastic" = ">= v1.0.0"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data elasticsearch` block:

```hcl
config data elasticsearch {
  # Optional string.
  # Default value:
  base_url = null

  # Optional string.
  # Default value:
  cloud_id = null

  # Optional string.
  # Default value:
  api_key_str = null

  # Optional list of string.
  # Default value:
  api_key = null

  # Optional string.
  # Default value:
  basic_auth_username = null

  # Optional string.
  # Default value:
  basic_auth_password = null

  # Optional string.
  # Default value:
  bearer_auth = null

  # Optional string.
  # Default value:
  ca_certs = null
}
```

## Usage

This data source accepts the following arguments within a `data elasticsearch` block:

```hcl
data elasticsearch {
  # Required string.
  #
  # For example:
  index = "some string"

  # Optional string.
  # Default value:
  id = null

  # Optional string.
  # Default value:
  query_string = null

  # Optional map of any single type.
  # Default value:
  query = null

  # Optional any type.
  # Default value:
  aggs = null

  # Optional bool.
  # Default value:
  only_hits = null

  # Optional list of string.
  # Default value:
  fields = null

  # Optional number.
  # Must be >= 0
  # Default value:
  size = 1000
}
```