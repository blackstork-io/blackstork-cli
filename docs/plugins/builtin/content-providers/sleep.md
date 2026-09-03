---
title: "`sleep` content provider"
plugin:
  name: blackstork/builtin
  description: "Pauses content generation for the specified duration, then renders a confirmation message. Intended for testing and debugging"
  tags: ["debug"]
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/builtin/"
resource:
  type: content-provider
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v1.0.0-rc1" "sleep" "content provider" >}}

## Description
Pauses content generation for the specified duration, then renders a confirmation message. Intended for testing and debugging.

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

