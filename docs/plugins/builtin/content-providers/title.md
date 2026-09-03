---
title: "`title` content provider"
plugin:
  name: blackstork/builtin
  description: "Renders a Go-templated string as a heading"
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

Renders a Go-templated string as a heading.

The final heading level must be from 1 through 6. Level 1 is the largest heading
(`<h1>` in HTML), and level 6 is the smallest (`<h6>`).


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

