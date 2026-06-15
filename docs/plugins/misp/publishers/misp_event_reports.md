---
title: "`misp_event_reports` publisher"
plugin:
  name: blackstork/misp
  description: "Publishes content to misp event reports"
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/misp/"
resource:
  type: publisher
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/misp" "misp" "v0.4.2" "misp_event_reports" "publisher" >}}

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `misp_event_reports` publisher locally via `blackstork-cli`, you must declare the `blackstork/misp` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/misp" = ">= v0.4.2"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Supported Formats

This publisher supports the delivery of documents processed by the following formatters:

- `md`

To specify the format, use the `format` argument inside the `publish` block to reference a specific `format` block or a formatter short name.


## Configuration

This publisher accepts the following configuration arguments within a `config publish misp_event_reports` block:

```hcl
config publish misp_event_reports {
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

This publisher accepts the following arguments within a `publish misp_event_reports` block:

```hcl
# Note: The `publish` block also accepts the generic `format` argument to link to a formatter.

publish misp_event_reports {
  # Required string.
  # Must be non-empty
  #
  # For example:
  event_id = "some string"

  # Required string.
  # Must be non-empty
  #
  # For example:
  name = "some string"

  # Optional string.
  # Must be one of: "0", "1", "2", "3", "4", "5"
  # Default value:
  distribution = null

  # Optional string.
  # Default value:
  sharing_group_id = null
}

```


