---
title: "`sleep` data source"
plugin:
  name: blackstork/builtin
  description: "Sleeps for the specified duration. Useful for testing and debugging"
  tags: ["debug"]
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/builtin/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v1.0.0-rc1" "sleep" "data source" >}}

## Description

Sleeps for the specified duration. Useful for testing and debugging.


The `sleep` data source is built into the BlackStork engine. It is available out-of-the-box
and requires no installation or dependency declaration.

## Configuration

This data source does not accept any configuration arguments.

## Usage

This data source accepts the following arguments within a `data sleep` block:

```hcl
data sleep {
  # Duration to sleep
  #
  # Optional string.
  # Must be non-empty
  # Default value:
  duration = "1s"
}
```