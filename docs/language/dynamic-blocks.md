---
title: Dynamic Blocks
description: Use dynamic blocks to iterate over data collections and programmatically generate repeating sections or content blocks in your BlackStork templates.
type: docs
weight: 71 
---

# Dynamic blocks

The `dynamic` block iterates over data collections to programmatically generate repeating document structures. 

Instead of manually defining a block for every item in a dataset, you pass a list to a `dynamic`
block. During evaluation, the engine loops through the list and duplicates the nested `section` or
`content` blocks for each item, injecting the specific item's data into the local evaluation
context.

The block signature requires only the `dynamic` keyword. It does not take a component name or a block name.

```hcl
section "example_section" {

  dynamic {
    # The list of items to iterate over
    items = query_jq(...)

    # The block(s) to duplicate for each item
    content text {
      value = "..."
    }
  }

}
```

You can define `dynamic` blocks directly inside a `document` block or nested within `section` blocks.

## Execution context

During evaluation, the `dynamic` block creates a local scope for each iteration. It injects two
specific variables into the evaluation context, making them available to all nested `vars`,
`content`, and `section` blocks:

- `.vars.dynamic_item` — The current element from the `items` list being processed.
- `.vars.dynamic_item_index` — The zero-based integer index of the current iteration.

Nested blocks inside the `dynamic` block evaluate exactly as if they were written out manually. You
can reference `.vars.dynamic_item` in Go templates or `query_jq()` functions to render data specific
to the current iteration.

## Supported arguments

- `items`: (required) A list of items to iterate over. This is typically populated using the
  `query_jq()` function to extract an array from the context. If the list is empty, the engine skips
  the nested blocks entirely.
- `depends_on`: (optional) A list of block identifiers wrapped in quotes (e.g.,
  `["content.text.first_paragraph"]`) that the engine must evaluate before executing this `dynamic`
  block.

## Supported nested blocks

- `content`: Defines the content components (e.g., `text`, `table`, `image`) to duplicate for each iteration.
- `section`: Defines structural sections to duplicate for each iteration.

**Note:** A `dynamic` block must contain at least one `content` or `section` block.

## Example

In the following example, a `dynamic` block iterates over a list of host data. For each host in the
array, it generates a new `section` containing a dynamic title and a `content.text` block, utilizing
the `.vars.dynamic_item` context.

```hcl
document "vulnerability_report" {

  vars {
    # A static array of objects for demonstration.
    # In production, this typically references a data block.
    vulnerable_hosts = [
      {
        ip       = "192.168.1.50"
        cve      = "CVE-2024-1234"
        severity = "High"
      },
      {
        ip       = "10.0.0.12"
        cve      = "CVE-2023-9876"
        severity = "Medium"
      }
    ]
  }

  section "findings" {
    title = "Host Findings"

    dynamic {
      # Pass the array to the dynamic block
      items = query_jq(".vars.vulnerable_hosts")

      # This section and its nested content are duplicated for each host
      section "host_detail" {

        # Access the current item via .vars.dynamic_item
        title = "Host: {{ .vars.dynamic_item.ip }}"

        content text {
          value = <<-EOT
            **Vulnerability:** {{ .vars.dynamic_item.cve }}
            **Severity:** {{ .vars.dynamic_item.severity }}
            **Index:** This is item #{{ .vars.dynamic_item_index }} in the findings array.
          EOT
        }
      }
    }
  }
}
```

During evaluation, the engine loops over the array and renders two distinct `host_detail` sections,
translating the JSON data into structured document elements.

## Next steps

See [Format blocks]({{< ref "format-blocks.md" >}}) to learn how to define document formats.

