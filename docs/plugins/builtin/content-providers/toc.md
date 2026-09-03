---
title: "`toc` content provider"
plugin:
  name: blackstork/builtin
  description: "Builds a table of contents from headings in the selected document scope. The result can be ordered or unordered and limited by heading level"
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/builtin/"
resource:
  type: content-provider
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v1.0.0-rc1" "toc" "content provider" >}}

## Description
Builds a table of contents from headings in the selected document scope. The result can be ordered or unordered and limited by heading level.

The `toc` content provider is built into the BlackStork engine. It is available out-of-the-box and requires no installation or dependency declaration.

## Configuration

This content provider does not accept any configuration arguments.

## Usage

This content provider accepts the following arguments within a `content toc` block:

```hcl
content toc {
  # Largest size of the header to be included the table of contents
  #
  # Optional integer.
  # Must be between 0 and 5 (inclusive)
  # Default value:
  start_level = 0

  # Smallest size of the header to be included in the table of contents
  #
  # Optional integer.
  # Must be between 0 and 5 (inclusive)
  # Default value:
  end_level = 2

  # Render as ordered list. If `false`, TOC is rendered as unordered list.
  #
  # Optional bool.
  # Default value:
  as_ordered_list = false

  # Scope for TOC to cover:
  #   "document" – collect headers in the document;
  #   "current" – collect headers in the current section or in the document, if TOC block is defined on the document's root level;
  #
  # Optional string.
  # Must be one of: "document", "current"
  # Default value:
  scope = "current"
}
```

