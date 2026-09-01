---
title: Evaluation Context
description: Understand the BlackStork evaluation context. Learn how to query data, use variables, accept inputs, and manipulate JSON structures with JQ.
type: docs
weight: 45
---

# Evaluation context

The evaluation context is the central data structure that holds all information available during a template evaluation. It contains the raw structured data returned by data sources, template metadata, inputs, and any manually defined variables.

Data inside the context is typically extracted or manipulated using the `query_jq()` function, and available for querying in Go templates.

## Keys

The context acts as a single JSON dictionary. You navigate this dictionary using standard [JQ JSON paths](https://jqlang.github.io/jq/manual/). 

The top-level root keys are:

- `.data` — Stores the structured JSON results returned by `data` blocks. The path format is `.data.<data-source-name>.<block-name>`.
- `.inputs` — Stores runtime input parameters (or their fallback default values). The path format is `.inputs.<input-name>`.
- `.document` — Contains metadata about the current document template. For example, `.document.meta` holds the attributes defined in the `meta` block at the root of the document.
- `.vars` — Stores the values of variables defined in the current block or inherited from parent blocks.

Depending on where a query is executed, the context may also contain localized metadata keys:

- `.section` — When evaluated inside a `section` block, this holds metadata specific to that section (e.g., `.section.meta`).
- `.content` — When evaluated inside a `content` block, this holds metadata specific to that piece of content (e.g., `.content.meta`).

## Inputs

The `input` block defines expected runtime parameters for the template. This is the primary method for making the templates dynamic and reusable across both `blackstork-cli` and the BlackStork SaaS platform.

```hcl
input "target_ip" {
  label         = "Target IP Address"
  type          = "string"
  default_value = "127.0.0.1"
}

input "include_history" {
  type          = "bool"
  default_value = true
}
```

The `input` blocks must be defined on the root level of the `document` block.

Once defined, inputs are injected into the evaluation context under the `.inputs` key. For example, `.inputs.target_ip` returns the value provided by the user at runtime, or `"127.0.0.1"` if none was supplied.

An input requires `type` and supports these attributes:

- `type` — One of `bool`, `datetime`, `json`, `number`, `secret`, or `string`.
- `label` — Text shown by the interactive CLI prompt.
- `description` — A description for people and tools that collect the input.
- `default_value` — The value used when the caller does not supply the input.
- `schema` — A JSON Schema used to validate a `json` input.

Pass CLI values with the repeatable `--input name=value` flag. Datetimes use RFC 3339. JSON values must be valid JSON. If no CLI value or default exists, the CLI prompts on standard input.


## Variables

You can define variables or construct new data structures inside the template using the `vars` block:

```hcl
vars {
  limit = 100

  # Variables can reference inputs
  target = query_jq(".inputs.target_ip")

  # Variables are evaluated in the order they are defined
  api_payload = {
    ip    = query_jq(".vars.target")
    limit = query_jq(".vars.limit")
  }
}
```

The `vars` block can be placed inside `document`, `section`, `content`, and reference blocks. The values become accessible in the context under the `.vars` key (e.g., `.vars.api_payload`).

### Local variable

For convenience, you can define a single, inline variable within a block using the `local_var` argument. This simply creates a variable named `local` in the current scope.

```hcl
content text {
  local_var = "World"
  value     = "Hello, {{ .vars.local }}!"
}
```

When evaluated, this block renders `Hello, World!`.

### Inheritance and Scope

Variables defined in a parent block (like `document` or a parent `section`) are automatically inherited by all nested blocks.

However, a nested block can redefine an inherited variable within its own `vars` block. This shadows the parent variable's value only within that specific nested scope.

```hcl
section {
  vars {
    foo = 11
    bar = 22
  }

  section {
    vars {
      foo = 33 # Shadows the parent 'foo'
      baz = 44
    }

    content text {
      # Renders: `Variable values: foo=33, bar=22, baz=44`
      value = "Variable values: foo={{ .vars.foo }}, bar={{ .vars.bar }}, baz={{ .vars.baz }}"
    }
  }
}
```

### Querying the context

To filter, extract, or mutate data within the context, use the `query_jq()` function.

This function accepts a string argument containing a standard [JQ expression](https://jqlang.github.io/jq/manual/), executes it against the current evaluation context, and returns the resulting JSON structure.

```hcl
section {
  vars {
    items = ["a", "b", "c"]
  }

  section {
    vars {
      items_count = query_jq(".vars.items | length")

      items_uppercase = query_jq(
        <<-EOT
          .vars.items | map(ascii_upcase) | join(":")
        EOT
      )
    }

    content text {
      # Renders: `Items count: 3; Uppercase items: A:B:C`
      value = "Items count: {{ .vars.items_count }}; Uppercase items: {{ .vars.items_uppercase }}"
    }
  }
}
```

## Next steps

See [Data Blocks]({{< ref "data-blocks.md" >}}) to learn how to fetch structured data from external sources into the evaluation context.
