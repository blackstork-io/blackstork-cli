---
title: Section Blocks
description: Group content and data into logical chapters, establish variable scopes, and iterate over arrays using section blocks.
type: docs
weight: 70
---

# Section blocks

The `section` block acts as a structural container within a document. It groups related data and content blocks, establishes a localized variable scope, and dictates the rendering hierarchy of the report.

Sections can be nested, when named, referenced across different templates to create reusable components (e.g., standard headers, disclaimers, or metric tables).


```hcl
# Root-level named section block
section "<section-name>" {
  # ...
}

document "soc-activity-overview" {

  section "exec-summary" {
    # ...
  }

  section "kpis" {

    section "slas" {
      # ...
    }

    section "coverage" {
      # ...
    }

    # ...
  }
}
```

If you define a `section` block at the root level of the configuration file (outside of a `document`), you must provide a section name. The combination of the block type (`section`) and the section name creates a unique identifier, allowing you to reference the block elsewhere.

If you define a `section` block directly within a `document` or another `section`, the section name is optional.

The engine evaluates and renders `section` blocks in the exact order they are declared.

## Supported arguments

- `title`: (optional) Sets a header for the section. This is syntactic sugar for a nested `content.title` block. During rendering, this title is placed at the very beginning of the section's output, preceding all other nested content or subsections.
- `local_var`: (optional) A shortcut to define a single local variable named `local` within the section's scope. See [Local variable]({{< ref "context.md#local-variable" >}}).
- `depends_on`: (optional) A list of content block traversals that must be evaluated first.
- `is_included`: (optional) A Boolean expression that controls whether the section is rendered.
- `required_vars`: (optional) A list of context paths that must exist before the section is evaluated.

## Supported nested blocks

- `meta`: (optional) Defines metadata for the section block. See [Metadata]({{< ref "configs.md#metadata" >}}).
- `vars`: (optional) Defines variables scoped to this section. Variables defined here are inherited by all nested content and subsections. See [Variables]({{< ref "context.md#variables" >}}).
- `content`: (optional) Defines the content rendered within this section. See [Content Blocks]({{< ref "content-blocks.md" >}}).
- `section`: (optional) Defines nested subsections to build deep document hierarchies.
- `dynamic`: (optional) Repeats nested content or sections for each item in a list.

## References

You can reuse a section block defined elsewhere in your configuration by using a `ref` block. This is highly recommended for standardizing components like headers, footers, and executive summaries across multiple reports. See [References]({{< ref "references.md" >}}) for details.

## Next steps

See [Dynamic blocks]({{< ref "dynamic-blocks.md" >}}) to learn how to iterate over data collections
to programmatically generate repeating sections.
