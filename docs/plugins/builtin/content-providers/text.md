---
title: "`text` content provider"
plugin:
  name: blackstork/builtin
  description: "Renders text block"
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/builtin/"
resource:
  type: content-provider
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v0.4.2" "text" "content provider" >}}

## Description
Renders text block

The `text` content provider is built into the BlackStork engine. It is available out-of-the-box and requires no installation or dependency declaration.

## Configuration

This content provider does not accept any configuration arguments.

## Usage

This content provider accepts the following arguments within a `content text` block:

```hcl
content text {
  # Text value rendered as a Go template
  #
  # Required string.
  #
  # For example:
  value = "Hello world!"
}
```

