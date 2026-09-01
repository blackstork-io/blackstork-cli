---
title: "`title` content provider"
plugin:
  name: blackstork/builtin
  description: "Produces a heading block"
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/builtin/"
resource:
  type: content-provider
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v1.0.0-rc1" "title" "content provider" >}}

## Description

Produces a heading block.

The title size after calculations must be in an interval [1; 6] inclusive, where 1
corresponds to the largest size (`<h1>`) and 6 corresponds to (`<h6>`)


The `title` content provider is built into the BlackStork engine. It is available out-of-the-box and requires no installation or dependency declaration.

## Configuration

This content provider does not accept any configuration arguments.

## Usage

This content provider accepts the following arguments within a `content title` block:

```hcl
content title {
  # Title content
  #
  # Required string.
  #
  # For example:
  value = "Vulnerability Report"

  # Sets the absolute size of the title. If `null` – absoulute title size is determined from the document structure.
  #
  # Optional integer.
  # Default value:
  absolute_size = null

  # Adjusts absolute size of the title. The value (which may be negative) is added to the `absolute_size` to produce the final title size.
  #
  # Optional integer.
  # Default value:
  relative_size = 0
}
```

