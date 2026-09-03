---
title: "`list` content provider"
plugin:
  name: blackstork/builtin
  description: "Renders a list of data items as an ordered or unordered list. Use `item_template` to control how each item appears"
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/builtin/"
resource:
  type: content-provider
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v1.0.0" "list" "content provider" >}}

## Description
Renders a list of data items as an ordered or unordered list. Use `item_template` to control how each item appears.

The `list` content provider is built into the BlackStork engine. It is available out-of-the-box and requires no installation or dependency declaration.

## Configuration

This content provider does not accept any configuration arguments.

## Usage

This content provider accepts the following arguments within a `content list` block:

```hcl
content list {
  # Go template for the item of the list
  #
  # Optional string.
  #
  # For example:
  # item_template = "[{{.Title}}]({{.URL}})"
  #
  # Default value:
  item_template = "{{.}}"

  # Optional string.
  # Must be one of: "unordered", "ordered"
  # Default value:
  format = "unordered"

  # List of items to render.
  #
  # Required list of jq queriable.
  #
  # For example:
  items = ["First item", "Second item", "Third item"]
}
```

