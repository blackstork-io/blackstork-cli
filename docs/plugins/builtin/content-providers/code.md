---
title: "`code` content provider"
plugin:
  name: blackstork/builtin
  description: "Renders a Go-templated string as a code block with an optional language for syntax highlighting"
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/builtin/"
resource:
  type: content-provider
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v1.0.0-rc1" "code" "content provider" >}}

## Description
Renders a Go-templated string as a code block with an optional language for syntax highlighting.

The `code` content provider is built into the BlackStork engine. It is available out-of-the-box and requires no installation or dependency declaration.

## Configuration

This content provider does not accept any configuration arguments.

## Usage

This content provider accepts the following arguments within a `content code` block:

```hcl
content code {
  # Required string.
  #
  # For example:
  value = "Text to be formatted as a code block"

  # Specifiy the code language for syntax highlighting
  #
  # Optional string.
  #
  # For example:
  # language = "json"
  #
  # Default value:
  language = "text"
}
```

