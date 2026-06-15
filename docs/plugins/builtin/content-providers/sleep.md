---
title: "`sleep` content provider"
plugin:
  name: blackstork/builtin
  description: "Sleeps for the specified duration. Useful for testing and debugging"
  tags: ["debug"]
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/builtin/"
resource:
  type: content-provider
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v0.4.2" "sleep" "content provider" >}}

## Description

Sleeps for the specified duration. Useful for testing and debugging.


The `sleep` content provider is built into the BlackStork engine. It is available out-of-the-box and requires no installation or dependency declaration.

## Configuration

This content provider does not accept any configuration arguments.

## Usage

This content provider accepts the following arguments within a `content sleep` block:

```hcl
content sleep {
  # Duration to sleep
  #
  # Optional string.
  # Must be non-empty
  # Default value:
  duration = "1s"
}
```

