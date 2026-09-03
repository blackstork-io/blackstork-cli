---
title: "`iris_cases` data source"
plugin:
  name: blackstork/iris
  description: "Retrieves cases from DFIR-IRIS. Supports filtering by case, customer, owner, severity, state, SOC, and opening date"
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/iris/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/iris" "iris" "v1.0.0" "iris_cases" "data source" >}}

## Description
Retrieves cases from DFIR-IRIS. Supports filtering by case, customer, owner, severity, state, SOC, and opening date.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `iris_cases` data source locally via `blackstork-cli`, you must declare the `blackstork/iris` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/iris" = ">= v1.0.0"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data iris_cases` block:

```hcl
config data iris_cases {
  # Iris API url
  #
  # Required string.
  # Must be non-empty
  #
  # For example:
  api_url = "some string"

  # Iris API Key
  #
  # Required string.
  # Must be non-empty
  #
  # For example:
  api_key = "some string"

  # Enable/disable insecure TLS
  #
  # Optional bool.
  # Default value:
  insecure = false
}
```

## Usage

This data source accepts the following arguments within a `data iris_cases` block:

```hcl
data iris_cases {
  # List of Case IDs
  #
  # Optional list of number.
  # Default value:
  case_ids = null

  # Case Customer ID
  #
  # Optional number.
  # Default value:
  customer_id = null

  # Case Owner ID
  #
  # Optional number.
  # Default value:
  owner_id = null

  # Case Severity ID
  #
  # Optional number.
  # Default value:
  severity_id = null

  # Case State ID
  #
  # Optional number.
  # Default value:
  state_id = null

  # Case SOC ID
  #
  # Optional string.
  # Default value:
  soc_id = null

  # Case opening date - lower boundary
  #
  # Optional string.
  # Default value:
  start_open_date = null

  # Case opening date - higher boundary
  #
  # Optional string.
  # Default value:
  end_open_date = null

  # Sort order
  #
  # Optional string.
  # Must be one of: "desc", "asc"
  # Default value:
  sort = "desc"

  # Size limit to retrieve
  #
  # Optional number.
  # Must be >= 0
  # Default value:
  size = 0
}
```