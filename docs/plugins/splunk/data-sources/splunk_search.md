---
title: "`splunk_search` data source"
plugin:
  name: blackstork/splunk
  description: "Runs a blocking Splunk search job and returns its results. Supports time bounds, result limits, status buckets, and required fields"
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/splunk/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/splunk" "splunk" "v1.0.0-rc1" "splunk_search" "data source" >}}

## Description
Runs a blocking Splunk search job and returns its results. Supports time bounds, result limits, status buckets, and required fields.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `splunk_search` data source locally via `blackstork-cli`, you must declare the `blackstork/splunk` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/splunk" = ">= v1.0.0-rc1"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data splunk_search` block:

```hcl
config data splunk_search {
  # Required string.
  #
  # For example:
  auth_token = "some string"

  # Optional string.
  # Default value:
  host = null

  # Optional string.
  # Default value:
  deployment_name = null
}
```

## Usage

This data source accepts the following arguments within a `data splunk_search` block:

```hcl
data splunk_search {
  # Required string.
  #
  # For example:
  search_query = "some string"

  # Optional number.
  # Default value:
  max_count = null

  # Optional number.
  # Default value:
  status_buckets = null

  # Optional list of string.
  # Default value:
  rf = null

  # Optional string.
  # Default value:
  earliest_time = null

  # Optional string.
  # Default value:
  latest_time = null
}
```