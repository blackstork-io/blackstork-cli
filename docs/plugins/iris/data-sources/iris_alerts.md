---
title: "`iris_alerts` data source"
plugin:
  name: blackstork/iris
  description: "Retrieves alerts from DFIR-IRIS. Supports filtering by alert, case, customer, owner, severity, classification, state, source, tags, and date range"
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/iris/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/iris" "iris" "v1.0.0-rc1" "iris_alerts" "data source" >}}

## Description
Retrieves alerts from DFIR-IRIS. Supports filtering by alert, case, customer, owner, severity, classification, state, source, tags, and date range.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `iris_alerts` data source locally via `blackstork-cli`, you must declare the `blackstork/iris` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/iris" = ">= v1.0.0-rc1"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data iris_alerts` block:

```hcl
config data iris_alerts {
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

This data source accepts the following arguments within a `data iris_alerts` block:

```hcl
data iris_alerts {
  # List of Alert IDs
  #
  # Optional list of number.
  # Default value:
  alert_ids = null

  # Alert Source
  #
  # Optional string.
  # Default value:
  alert_source = null

  # List of tags
  #
  # Optional list of string.
  # Default value:
  tags = null

  # Case ID
  #
  # Optional number.
  # Default value:
  case_id = null

  # Alert Customer ID
  #
  # Optional number.
  # Default value:
  customer_id = null

  # Alert Owner ID
  #
  # Optional number.
  # Default value:
  owner_id = null

  # Alert Severity ID
  #
  # Optional number.
  # Default value:
  severity_id = null

  # Alert Classification ID
  #
  # Optional number.
  # Default value:
  classification_id = null

  # Alert State ID
  #
  # Optional number.
  # Default value:
  status_id = null

  # Alert Date - lower boundary
  #
  # Optional string.
  # Default value:
  alert_start_date = null

  # Alert Date - higher boundary
  #
  # Optional string.
  # Default value:
  alert_end_date = null

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