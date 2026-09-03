---
title: "`microsoft_sentinel_incidents` data source"
plugin:
  name: blackstork/microsoft
  description: "Retrieves incidents from a Microsoft Sentinel workspace. Supports OData filtering, ordering, and result limits"
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/microsoft/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/microsoft" "microsoft" "v1.0.0" "microsoft_sentinel_incidents" "data source" >}}

## Description
Retrieves incidents from a Microsoft Sentinel workspace. Supports OData filtering, ordering, and result limits.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `microsoft_sentinel_incidents` data source locally via `blackstork-cli`, you must declare the `blackstork/microsoft` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/microsoft" = ">= v1.0.0"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data microsoft_sentinel_incidents` block:

```hcl
config data microsoft_sentinel_incidents {
  # The Azure client ID
  #
  # Required string.
  #
  # For example:
  client_id = "some string"

  # The Azure client secret
  #
  # Required string.
  #
  # For example:
  client_secret = "some string"

  # The Azure tenant ID
  #
  # Required string.
  #
  # For example:
  tenant_id = "some string"

  # The Azure subscription ID
  #
  # Required string.
  #
  # For example:
  subscription_id = "some string"

  # The Azure resource group name
  #
  # Required string.
  #
  # For example:
  resource_group_name = "some string"

  # The Azure workspace name
  #
  # Required string.
  #
  # For example:
  workspace_name = "some string"
}
```

## Usage

This data source accepts the following arguments within a `data microsoft_sentinel_incidents` block:

```hcl
data microsoft_sentinel_incidents {
  # The filter expression
  #
  # Optional string.
  # Default value:
  filter = null

  # Number of objects to be returned
  #
  # Optional number.
  # Must be >= 1
  # Default value:
  size = 50

  # The order by expression
  #
  # Optional string.
  # Default value:
  order_by = null
}
```