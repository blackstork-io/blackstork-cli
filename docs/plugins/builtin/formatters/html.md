---
title: "`html` formatter"
plugin:
  name: blackstork/builtin
  description: "Formats content in HTML"
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/builtin/"
resource:
  type: formatter
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v0.4.2" "html" "formatter" >}}

The `html` formatter is built into the BlackStork engine. It is available out-of-the-box and requires no installation or dependency declaration.

## Configuration

This formatter accepts the following configuration arguments within a `config format html` block:

```hcl
config format html {
  # Required string.
  # Must be non-empty
  #
  # For example:
  page_template = "some string"
}
```

## Usage

This formatter accepts the following arguments within a `format html` block:

```hcl
format html {
  # HTML templates for document, section, and content blocks. For content blocks, the key must include the content provider.
  #
  # Optional map of any single type.
  #
  # For example:
  # template_per_type = {
  #   "content.image" = "<img src=\"{{ .src }}\" alt=\"{{ .alt }}\" class=\"img-w-10\" />"
  #   "content.text"  = "<span class=\"text-block\">{{ .value }}</span>"
  # }
  #
  # Default value:
  template_per_type = null

  # HTML templates for specific named blocks.
  #
  # Optional map of any single type.
  #
  # For example:
  # template_per_block = {
  #   "content.text.foo" = "<span class=\"text-block\">{{ .block.value }}</span>"
  #   "section.bar"      = "<h1>{{ .title }}</h1><p>{{ .content }}</p>"
  # }
  #
  # Default value:
  template_per_block = null

  # CSS code to include inline
  #
  # Optional string.
  # Default value:
  css_inline = null

  # CSS code to include inline inside <style> tag of type `text/tailwindcss`
  #
  # Optional string.
  # Default value:
  css_inline_tailwind = null

  # JS code to include inline
  #
  # Optional string.
  # Default value:
  js_inline = null

  # CSS source URLs
  #
  # Optional list of string.
  # Default value:
  css_sources = null

  # JS source URLs
  #
  # Optional list of string.
  # Default value:
  js_sources = null
}
```

