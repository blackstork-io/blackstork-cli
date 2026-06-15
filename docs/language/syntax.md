---
title: Syntax
description: Syntax rules and structural components of the BlackStork configuration language. Learn how to use blocks, arguments, and comments.
type: docs
weight: 10
---

# Syntax

This page describes the native syntax of the BlackStork configuration language. Built on top of the
[HashiCorp Configuration Language](https://github.com/hashicorp/hcl/blob/main/hclsyntax/spec.md)
(HCL), the syntax is composed of two fundamental elements: **arguments** and **blocks**.

## Arguments

Arguments assign values to specific names within a block.

```hcl
content text {
  value = "An example of the text value"
}
```

The argument identifier (`value` in the snippet above) can contain letters, digits, underscores (`_`), and hyphens (`-`). The first character of an identifier must not be a digit.

## Blocks

Blocks act as containers. They define plugin configurations, data queries, content structures, variables, and output destinations.

```hcl
document "test_document" {

  data elasticsearch "es-events" {
    index = "events"
  }

  content text {
    value = "My custom static text"
  }

  vars {
    foo = "xyz"
  }
}
```

Every block has a type (for example, `document`, `data` or `content`) that dictates its signature and allowed arguments. Depending on the type and context, a block may require a string label (such as `"es-events"`) or remain anonymous.

Block types are categorized by their function in the rendering pipeline:

- [Configuration]({{< ref "configs.md" >}}): `blackstork`, `config`, and `meta` blocks
- [Documents]({{< ref "documents.md" >}}): `document` block
- [Data definitions]({{< ref "data-blocks.md" >}}): `data` block
- [Content definitions]({{< ref "content-blocks.md" >}}): `content` block
- [Content structure]({{< ref "section-blocks.md" >}}): `section` block
- [Formatting]({{< ref "format-blocks.md" >}}): `format` block
- [Publishing]({{< ref "publish-blocks.md" >}}): `publish` block
- [References]({{< ref "references.md" >}}): Reusing blocks via reference

## Comments

The configuration language supports three comment styles:

- `#` begins a single-line comment, terminating at the end of the line.
- `//` is an alternative single-line comment syntax.
- `/*` and `*/` define multi-line block comments.

The `#` symbol is the standard idiomatic comment style for BlackStork configurations. Future formatting tools will standardize on `#` for single-line comments.

## Character encoding

BlackStork configuration files must be UTF-8 encoded. Non-ASCII characters are supported in comments and string values.

## Next steps

See [Configuration]({{< ref "configs.md" >}}) to learn how to configure the engine and set up external integrations.
