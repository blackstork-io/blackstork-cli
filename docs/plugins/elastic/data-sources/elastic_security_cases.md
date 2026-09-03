---
title: "`elastic_security_cases` data source"
plugin:
  name: blackstork/elastic
  description: "Retrieves Elastic Security cases from Kibana. Supports filtering, searching, sorting, and limiting the returned cases"
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/elastic/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/elastic" "elastic" "v1.0.0" "elastic_security_cases" "data source" >}}

## Description
Retrieves Elastic Security cases from Kibana. Supports filtering, searching, sorting, and limiting the returned cases.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `elastic_security_cases` data source locally via `blackstork-cli`, you must declare the `blackstork/elastic` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/elastic" = ">= v1.0.0"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data elastic_security_cases` block:

```hcl
config data elastic_security_cases {
  # Required string.
  #
  # For example:
  kibana_endpoint_url = "some string"

  # Optional string.
  # Default value:
  api_key_str = null

  # Optional [string, string].
  # Default value:
  api_key = null
}
```

## Usage

This data source accepts the following arguments within a `data elastic_security_cases` block:

```hcl
data elastic_security_cases {
  # Optional string.
  # Default value:
  space_id = null

  # Optional list of string.
  # Default value:
  assignees = null

  # Optional string.
  # Default value:
  default_search_operator = null

  # Optional string.
  # Default value:
  from = null

  # Optional list of string.
  # Default value:
  owner = null

  # Optional list of string.
  # Default value:
  reporters = null

  # Optional string.
  # Default value:
  search = null

  # Optional list of string.
  # Default value:
  search_fields = null

  # Optional string.
  # Default value:
  severity = null

  # Optional string.
  # Default value:
  sort_field = null

  # Optional string.
  # Default value:
  sort_order = null

  # Optional string.
  # Default value:
  status = null

  # Optional list of string.
  # Default value:
  tags = null

  # Optional string.
  # Default value:
  to = null

  # Optional number.
  # Default value:
  size = null
}
```