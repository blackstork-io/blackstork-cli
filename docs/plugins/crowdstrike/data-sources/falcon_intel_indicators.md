---
title: "`falcon_intel_indicators` data source"
plugin:
  name: blackstork/crowdstrike
  description: "The `falcon_intel_indicators` data source fetches intel indicators from Falcon API"
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/crowdstrike/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/crowdstrike" "crowdstrike" "v0.4.2" "falcon_intel_indicators" "data source" >}}

## Description
The `falcon_intel_indicators` data source fetches intel indicators from Falcon API.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `falcon_intel_indicators` data source locally via `blackstork-cli`, you must declare the `blackstork/crowdstrike` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/crowdstrike" = ">= v0.4.2"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data falcon_intel_indicators` block:

```hcl
config data falcon_intel_indicators {
  # Client ID for accessing CrowdStrike Falcon Platform
  #
  # Required string.
  # Must be non-empty
  #
  # For example:
  client_id = "some string"

  # Client Secret for accessing CrowdStrike Falcon Platform
  #
  # Required string.
  # Must be non-empty
  #
  # For example:
  client_secret = "some string"

  # Member CID for MSSP
  #
  # Optional string.
  # Default value:
  member_cid = null

  # Falcon cloud abbreviation
  #
  # Optional string.
  # Must be one of: "autodiscover", "us-1", "us-2", "eu-1", "us-gov-1", "gov1"
  #
  # For example:
  # client_cloud = "us-1"
  #
  # Default value:
  client_cloud = null
}
```

## Usage

This data source accepts the following arguments within a `data falcon_intel_indicators` block:

```hcl
data falcon_intel_indicators {
  # limit the number of queried items
  #
  # Optional integer.
  # Default value:
  limit = 10

  # Indicators filter expression using Falcon Query Language (FQL)
  #
  # Optional string.
  # Default value:
  filter = null

  # Indicators sort expression using Falcon Query Language (FQL)
  #
  # Optional string.
  # Default value:
  sort = null
}
```