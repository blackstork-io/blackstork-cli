---
title: Documents
description: Learn how to use document blocks to define report templates, data requirements, and content structures in the BlackStork configuration language.
type: docs
weight: 40
---

# Documents

The `document` block is the primary container for a report template. It encapsulates the data requirements, variables, content layout, formatting and publishing instructions for a specific document.


```hcl
document "<document-name>" {

  title = "<document title>"

  # ...
}
```

A `document` block must be defined at the root level of a configuration file. The `<document-name>` serves as a unique identifier within your project. When using `blackstork-cli`, this name is passed to the `render` command to specify which template to evaluate.

## Supported arguments

- `title`: (optional) The top-level header of the document. This is syntactic sugar for a nested `content.title` block. During rendering, this title is placed at the very beginning of the document, preceding all other content or section blocks.

## Supported blocks

- [`meta`]({{< ref "configs.md#metadata" >}}): Descriptive metadata for the template, such as author, version, and tags.
- [`input`]({{< ref "context.md#inputs" >}}): (optional) Defines the required parameters for the template. Using `input` blocks is the recommended way to handle dynamic values (like incident IDs or date ranges) to ensure templates are portable between the CLI and the BlackStork SaaS platform.
- [`vars`]({{< ref "context.md#variables" >}}): (optional) Defines static variables or internal data structures for use within the template.
- [`data`]({{< ref "data-blocks.md" >}}): Specifies the integrations used to fetch structured data from external sources.
- [`content`]({{< ref "content-blocks.md" >}}): Defines individual blocks of text, tables, or AI-generated narratives.
- [`section`]({{< ref "section-blocks.md" >}}): Organizes content and data blocks into logical groups or chapters.
- [`format`]({{< ref "format-blocks.md" >}}): Instructions for formatting the document.
- [`publish`]({{< ref "publish-blocks.md" >}}): Instructions for delivering the rendered document to specific destinations.

## Next steps

See [Evaluation Context]({{< ref "context.md" >}}) to learn how the data fetched in these blocks is
stored and accessed during the rendering process.
