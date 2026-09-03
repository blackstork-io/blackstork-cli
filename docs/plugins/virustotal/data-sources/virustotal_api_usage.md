---
title: "`virustotal_api_usage` data source"
plugin:
  name: blackstork/virustotal
  description: "Retrieves VirusTotal API usage statistics for a user or group, optionally limited to a date range"
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/virustotal/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/virustotal" "virustotal" "v1.0.0" "virustotal_api_usage" "data source" >}}

## Description
Retrieves VirusTotal API usage statistics for a user or group, optionally limited to a date range.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `virustotal_api_usage` data source locally via `blackstork-cli`, you must declare the `blackstork/virustotal` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/virustotal" = ">= v1.0.0"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data virustotal_api_usage` block:

```hcl
config data virustotal_api_usage {
  # Required string.
  # Must be non-empty
  #
  # For example:
  api_key = "some string"
}
```

## Usage

This data source accepts the following arguments within a `data virustotal_api_usage` block:

```hcl
data virustotal_api_usage {
  # Optional string.
  # Default value:
  user_id = null

  # Optional string.
  # Default value:
  group_id = null

  # Optional string.
  # Default value:
  start_date = null

  # Optional string.
  # Default value:
  end_date = null
}
```