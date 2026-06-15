---
title: "`image` content provider"
plugin:
  name: blackstork/builtin
  description: "Renders an image"
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/builtin/"
resource:
  type: content-provider
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v0.4.2" "image" "content provider" >}}

## Description
Renders an image

The `image` content provider is built into the BlackStork engine. It is available out-of-the-box and requires no installation or dependency declaration.

## Configuration

This content provider does not accept any configuration arguments.

## Usage

This content provider accepts the following arguments within a `content image` block:

```hcl
content image {
  # Required string.
  # Must be non-empty
  #
  # For example:
  src = "https://example.com/img.png"

  # Optional string.
  #
  # For example:
  # alt = "Text description of the image"
  #
  # Default value:
  alt = ""
}
```

