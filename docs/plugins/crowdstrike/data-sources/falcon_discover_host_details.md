---
title: "`falcon_discover_host_details` data source"
plugin:
  name: blackstork/crowdstrike
  description: "Retrieves host details from CrowdStrike Falcon Discover, optionally filtered with Falcon Query Language (FQL). Returns a list of host records"
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/crowdstrike/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/crowdstrike" "crowdstrike" "v1.0.0" "falcon_discover_host_details" "data source" >}}

## Description
Retrieves host details from CrowdStrike Falcon Discover, optionally filtered with Falcon Query Language (FQL). Returns a list of host records.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `falcon_discover_host_details` data source locally via `blackstork-cli`, you must declare the `blackstork/crowdstrike` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/crowdstrike" = ">= v1.0.0"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data falcon_discover_host_details` block:

```hcl
config data falcon_discover_host_details {
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

This data source accepts the following arguments within a `data falcon_discover_host_details` block:

```hcl
data falcon_discover_host_details {
  # limit the number of queried items
  #
  # Optional integer.
  # Default value:
  limit = 10

  # Host search expression using Falcon Query Language (FQL)
  #
  # Optional string.
  # Default value:
  filter = null
}
```