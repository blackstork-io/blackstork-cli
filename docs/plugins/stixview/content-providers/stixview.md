---
title: "`stixview` content provider"
plugin:
  name: blackstork/stixview
  description: ""
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/stixview/"
resource:
  type: content-provider
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/stixview" "stixview" "v1.0.0-rc1" "stixview" "content provider" >}}

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `stixview` content provider locally via `blackstork-cli`, you must declare the `blackstork/stixview` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/stixview" = ">= v1.0.0-rc1"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This content provider does not accept any configuration arguments.

## Usage

This content provider accepts the following arguments within a `content stixview` block:

```hcl
content stixview {
  # Optional string.
  # Default value:
  gist_id = null

  # Optional string.
  # Default value:
  stix_url = null

  # Optional string.
  # Default value:
  caption = null

  # Optional bool.
  # Default value:
  show_footer = null

  # Optional bool.
  # Default value:
  show_sidebar = null

  # Optional bool.
  # Default value:
  show_tlp_as_tags = null

  # Optional bool.
  # Default value:
  show_marking_nodes = null

  # Optional bool.
  # Default value:
  show_labels = null

  # Optional bool.
  # Default value:
  show_idrefs = null

  # Optional number.
  # Default value:
  width = null

  # Optional number.
  # Default value:
  height = null

  # Optional jq queriable.
  # Default value:
  objects = null
}
```

