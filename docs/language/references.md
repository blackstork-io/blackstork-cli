---
title: References
description: Learn how to reuse BlackStork blocks across your templates using block references.
type: docs
weight: 80
---

# References

The BlackStork configuration language supports code reuse through block references. You can reference any named `data`, `content`, `section`, `format`, or `publish` block defined at the root level of your configuration files.

To include a root-level block into a document or section, use the `ref` keyword in place of the component name, and specify the target block using the `base` argument.

```hcl
content <content-provider> "<block-name>" {
  # ...
}

data <data-source> "<block-name>" {
  # ...
}

section "<block-name>" {
  # ...
}

format <formatter-name> "<block-name>" {
  # ...
}

document "example" {

  # Named reference
  <block-type> ref "<new-block-name>" {
    base = <block-identifier>
    # ...
  }

  # Anonymous reference
  content ref {
    base = content.<content-provider>.<block-name>
  }

}
```

- `<block-type>` must be `data`, `content`, `section`, `format`, or `publish`.
- `<block-identifier>` is the absolute path to the referenced block. It consists of a dot-separated list of the components in the target block's signature (for example, `content.text.greeting` or `section.executive_summary`). Do not wrap the block name in quotes within the identifier.

If you define a `ref` block at the root level of a file, you must provide a block name. If you define it within a `document` or `section` block, the block name is optional.

{{< hint warning >}}
**Anonymous Reference Name Collisions**

An anonymous `ref` block automatically inherits the block name of its referenced `base` block. Because block signatures must be unique within the same scope, declaring multiple anonymous references to the same base block will cause an override collision. To reference the same base block multiple times within the same scope, explicitly assign a unique block name to each `ref` block.
{{< /hint >}}

## Overriding arguments

A `ref` block can include arguments that override the values defined in the base block. This allows you to define a standard component once (such as a specifically styled text block or a standard data query) and adjust its behavior for specific documents.

For example:

```hcl
# Define the base block at the root level
content text "hello_world" {
  value = "Hello, World!"
}

document "example" {

  # Reference the base block and override the 'value' argument
  content ref "hello_john" {
    base  = content.text.hello_world
    value = "Hello, John!"
  }

  # Anonymous reference overriding the 'value' argument
  content ref {
    base  = content.text.hello_world
    value = "Hello, New World!"
  }

}
```

## Next steps

See [HCL Expressions]({{< ref "hcl.md" >}}) to learn which HashiCorp Configuration Language (HCL) expressions are supported in BlackStork templates and how to use them to construct dynamic arguments.
