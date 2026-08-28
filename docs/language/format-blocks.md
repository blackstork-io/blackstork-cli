---
title: Format Blocks
description: Use format blocks to define the visual presentation, styling, and structural overrides for rendered BlackStork documents.
type: docs
weight: 72
---

# Format blocks

The `format` block defines the visual presentation, styling, and layout of the rendered document. 

In BlackStork, report generation is divided into two distinct phases: data evaluation and document rendering. While `data` and `content` blocks handle extracting and structuring information, `format` blocks dictate exactly how that structured data is presented to the end user.

The block signature requires the name of the formatter that will execute the block.

```hcl
# Root-level definition of a format block
format <formatter-name> "<block-name>" {
  # ...
}

document "example" {

  # In-document named definition of a format block
  format <formatter-name> "<block-name>" {
    # ...
  }

  # In-document anonymous definition of a format block
  format <formatter-name> {
    # ...
  }

}
```

If you define a `format` block at the root level of the file (outside of a `document` block), you must provide a block name. The combination of block type (`format`), formatter name, and block name creates a unique identifier. If you define the block directly within a `document`, the block name is optional.

A single `document` block can contain multiple `format` blocks. During execution, the engine generates a formatted output for a requested `publish` block using a corresponding defined format block.

Within a `document`, `format` blocks can be defined only on the root level.

See [Formatters]({{< ref "formatters.md" >}}) for the list of supported formatters.

## Execution context

Format blocks operate on the *evaluated* document tree. When the formatter runs, the engine has already executed all `query_jq()` functions and resolved all Go templates inside your `content` blocks.

Templates inside the format blocks receive full data context with extras. The extras depend on the formatter and might include rendered output from content blocks as Markdown or HTML.

## Supported arguments

A `format` block accepts both generic arguments and arguments specific to the selected formatter.

### Generic arguments

- `config`: (optional) A string referencing a named `config` block. If provided, the engine uses this configuration instead of the default configuration for the formatter. See [Block configuration]({{< ref "configs.md#block-configuration" >}}) for details.

### Formatter arguments

Arguments determining styling, layout overrides, and external assets depend strictly on the formatter. For example, the `html` formatter provides arguments like `template_per_type` and `template_per_block` to selectively override how specific components are rendered. Refer to the specific [Formatters]({{< ref "formatters.md" >}}) documentation for supported arguments.

## Supported blocks

- `meta`: (optional) Defines metadata for the format block. See [Metadata]({{< ref "configs.md#metadata" >}}).
- `config`: (optional) Defines an inline configuration for the block. If provided, this block takes highest precedence, overriding both the `config` argument and the default configuration for the formatter.

## Example

By default, BlackStork translates your content tree into standard semantic Markdown. You can use formatter arguments to override this default behavior. 

In the following example, the `html` formatter injects Tailwind CSS, wraps the entire document in a styled `<main>` container, and explicitly overrides the rendering of the `executive_summary` block.

```hcl
document "threat_brief" {

  # 1. Content Definition (The Data Model)
  content text "executive_summary" {
    value = "Ransomware activity increased by 15% across the financial sector."
  }

  content table "metrics" {
    # ... table definition ...
  }

  # 2. Format Definition (The View)
  format html "styled_report" {

    # Inject external assets (e.g., a CSS framework)
    js_sources = [
      "https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4"
    ]

    css_inline_tailwind = <<-EOT
      body { @apply bg-slate-100 text-slate-800 antialiased font-sans; }
    EOT

    # Override rendering for ALL blocks of a specific type (e.g., the document root)
    template_per_type = {
      "document" = <<-EOT
        <main class="max-w-4xl mx-auto py-12 px-8 bg-white shadow-xl mt-10 rounded-2xl">
          <!-- .content contains the evaluated HTML of all child blocks -->
          {{ .content }}
        </main>
      EOT
    }

    # Override rendering for a SPECIFIC named block
    template_per_block = {
      "content.text.executive_summary" = <<-EOT
        <div class="border-l-4 border-blue-600 pl-5 py-3 my-6 bg-blue-50 text-blue-900 rounded-r-md text-lg font-medium">
          <!-- .value_html contains the HTML-formatted output of the block -->
          {{ .value_html }}
        </div>
      EOT
    }
  }
}
```

In this workflow, the `content.text.executive_summary` block renders the text while the `format.html.styled_report` block intercepts the evaluated output at the end of the pipeline and wraps it in the specified visual layout.

## Next steps

See [Publish Blocks]({{< ref "publish-blocks.md" >}}) to learn how to route and deliver the formatted document to external destinations.
