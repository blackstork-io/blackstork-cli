---
title: "`md` formatter"
plugin:
  name: blackstork/builtin
  description: "Formats content in Markdown"
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/builtin/"
resource:
  type: formatter
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v1.0.0-rc1" "md" "formatter" >}}

The `md` formatter is built into the BlackStork engine. It is available out-of-the-box and requires no installation or dependency declaration.

## Configuration

This formatter does not accept any configuration arguments.

## Usage

This formatter accepts the following arguments within a `format md` block:

```hcl
format md {
  # Arbitrary key-value map to be put in the frontmatter
  #
  # Optional jq queriable.
  #
  # For example:
  # frontmatter = {
  #   key = "arbitrary value"
  #   key2 = {
  #     nested_key = 42
  #   }
  # }
  #
  # Default value:
  frontmatter = null

  # Format of the frontmatter.
  #
  # Optional string.
  # Must be one of: "yaml", "toml", "json"
  # Default value:
  frontmatter_format = "yaml"
}
```

