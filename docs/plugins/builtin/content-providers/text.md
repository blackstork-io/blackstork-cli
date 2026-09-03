---
title: "`text` content provider"
plugin:
  name: blackstork/builtin
  description: "Renders a Go-templated string as document text"
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/builtin/"
resource:
  type: content-provider
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v1.0.0-rc1" "text" "content provider" >}}

## Description
Renders a Go-templated string as document text.

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

