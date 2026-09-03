---
title: "`misp_event_reports` publisher"
plugin:
  name: blackstork/misp
  description: "Publishes a Markdown document as an Event Report attached to an existing MISP event."
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/misp/"
resource:
  type: publisher
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/misp" "misp" "v1.0.0-rc1" "misp_event_reports" "publisher" >}}

## Description
Publishes a Markdown document as an Event Report attached to an existing MISP event.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `misp_event_reports` publisher locally via `blackstork-cli`, you must declare the `blackstork/misp` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/misp" = ">= v1.0.0-rc1"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Supported Formatters

This publisher supports the delivery of documents formatted by the following formatters:

- `md`

Use `format_ref` in the `publish` block to reference a root-level `format` block or a format block in the current document. References to in-document blocks require the `document.` prefix.


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
# The `publish` block also accepts the generic `format_ref` argument to link to a formatter.

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

