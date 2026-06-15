---
title: "`misp_events` data source"
plugin:
  name: blackstork/misp
  description: "The `misp_events` data source fetches MISP events"
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/misp/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/misp" "misp" "v0.4.2" "misp_events" "data source" >}}

## Description
The `misp_events` data source fetches MISP events

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `misp_events` data source locally via `blackstork-cli`, you must declare the `blackstork/misp` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/misp" = ">= v0.4.2"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data misp_events` block:

```hcl
config data misp_events {
  # misp api key
  #
  # Required string.
  # Must be non-empty
  #
  # For example:
  api_key = "some string"

  # misp base url
  #
  # Required string.
  # Must be non-empty
  #
  # For example:
  base_url = "some string"

  # skip ssl verification
  #
  # Optional bool.
  # Default value:
  skip_ssl = false
}
```

## Usage

This data source accepts the following arguments within a `data misp_events` block:

```hcl
data misp_events {
  # Required string.
  #
  # For example:
  value = "some string"

  # Optional string.
  # Default value:
  type = null

  # Optional string.
  # Default value:
  category = null

  # Optional string.
  # Default value:
  org = null

  # Optional list of string.
  # Default value:
  tags = null

  # Optional list of string.
  # Default value:
  event_tags = null

  # Optional string.
  # Default value:
  searchall = null

  # Optional string.
  # Default value:
  from = null

  # Optional string.
  # Default value:
  to = null

  # Optional string.
  # Default value:
  last = null

  # Optional number.
  # Default value:
  event_id = null

  # Optional bool.
  # Default value:
  with_attachments = null

  # Optional list of string.
  # Default value:
  sharing_groups = null

  # Optional bool.
  # Default value:
  only_metadata = null

  # Optional string.
  # Default value:
  uuid = null

  # Optional bool.
  # Default value:
  include_sightings = null

  # Optional number.
  # Default value:
  threat_level_id = null

  # Optional number.
  # Default value:
  limit = 10
}
```