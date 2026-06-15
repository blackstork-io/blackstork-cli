---
title: "`blockquote` content provider"
plugin:
  name: blackstork/builtin
  description: "Formats text as a blockquote"
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/builtin/"
resource:
  type: content-provider
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v0.4.2" "blockquote" "content provider" >}}

## Description
Formats text as a blockquote

The `blockquote` content provider is built into the BlackStork engine. It is available out-of-the-box and requires no installation or dependency declaration.

## Configuration

This content provider does not accept any configuration arguments.

## Usage

This content provider accepts the following arguments within a `content blockquote` block:

```hcl
content blockquote {
  # Required string.
  #
  # For example:
  value = "Text to be formatted as a quote"
}
```

