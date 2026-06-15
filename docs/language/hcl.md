---
title: HCL Expressions
description: Learn how to use native HashiCorp Configuration Language (HCL) expressions to dynamically compute values, evaluate conditionals, and transform data in BlackStork templates.
type: docs
weight: 90
---

# HCL expressions

The BlackStork configuration language supports native [HashiCorp Configuration Language (HCL)](https://github.com/hashicorp/hcl/blob/main/hclsyntax/spec.md#expressions) expressions.

HCL expressions allow you to dynamically compute values, evaluate conditionals, and transform collections (lists and objects) directly within your block arguments.

Use HCL expressions to construct dynamic arguments for plugins, or to transform static data inside a `vars` block before it is processed by the rendering engine.

## Execution phase

It is important to distinguish HCL expressions from the other dynamic languages used in BlackStork:

- **HCL Expressions (`${...}`, `[for ...]`)**: Evaluated by the parser when the configuration is loaded. They construct the raw arguments passed to blocks.
- **JQ Queries (`query_jq(...)`)**: Evaluated during the data pipeline phase against the BlackStork evaluation context. They extract and filter structured data.
- **Go Templates (`{{ ... }}`)**: Evaluated during the rendering phase. They inject evaluated context data into the final string outputs (like Markdown or HTML).

## Supported expressions

BlackStork supports the full suite of HCL expressions, including:

- **String interpolation:** Evaluate expressions inside strings using the `${...}` syntax.
- **Arithmetic and logic:** Use standard mathematical operators (`+`, `-`, `*`, `/`, `%`) and logical operators (`&&`, `||`, `!`).
- **Conditionals:** Choose between two values based on a boolean expression using the `<CONDITION> ? <TRUE_VAL> : <FALSE_VAL>` syntax.
- **`for` expressions:** Iterate over lists, tuples, maps, and objects to filter elements or construct new collections.
- **Splat expressions:** Extract a specific attribute from every object in a list using the `[*]` syntax.
- **Template directives:** Use `%{~ if ... ~}` and `%{~ for ... ~}` inside heredoc strings for complex, multi-line string construction.

{{< hint warning >}}
**Use Native BlackStork Features**

While the underlying parser currently evaluates HCL string template directives (such as `%{~ if ... ~}` and `%{~ for ... ~}`), they are not considered a core feature of the BlackStork configuration language. 

We strongly recommend using BlackStork's native capabilities - specifically **Go templates** inside
content blocks and **JQ queries** for data transformation - rather than relying on HCL string
directives. Support for HCL-specific template directives may be removed in a future release.
{{< /hint >}}

## Example

The following configuration demonstrates how various HCL expressions can be used within a `vars` block to compute data, which is then rendered by a `content.text` block.


```hcl
document "example" {

  vars {

    arithmetic = "1 + 2 = ${1 + 2}"
    # "arithmetic": "1 + 2 = 3",

    logic = "true and false is ${true && false}"
    # "logic": "true and false is false",

    conditionals = "2 is ${2 % 2 == 0 ? "even" : "odd"}"
    # "conditionals": "2 is even"

    # technically, this is a tuple
    loop_over_list = [ for el in [1, 2, 3]: el * 2 ]
    # "loop_over_list": [
    #     2,
    #     4,
    #     6
    # ],

    loop_over_tuple = [ for el in [1, "two", 3]: "value is ${el}" ]
    # "loop_over_tuple": [
    #     "value is 1",
    #     "value is two",
    #     "value is 3"
    # ],

    # technically, this is an object
    loop_over_map = [ for k, v in {"a": 1, "b": 2, "c": 3}: "key ${k}: value ${v}" ]
    # "loop_over_map": [
    #     "key a: value 1",
    #     "key b: value 2",
    #     "key c: value 3"
    # ],

    loop_over_object = [ for k, v in {"a": 1, "b": "two", "c": 3}: "key ${k}: value ${v}" ]
    # "loop_over_object": [
    #     "key a: value 1",
    #     "key b: value two",
    #     "key c: value 3"
    # ],

    loop_creating_object = { for v in [1, 2, 3]: "${v}" => v * 2 }
    # "loop_creating_object": {
    #     "1": 2,
    #     "2": 4,
    #     "3": 6
    # },

    loop_with_filter = { for v in [1, 2, 3, 4]: "${v}" => v * 2 if v % 2 == 0 }
    # "loop_with_filter": {
    #     "2": 4,
    #     "4": 8
    # },

    loop_with_grouping = { for v in [1, 2, 3, 4]: (v%2 == 0 ? "evens" : "odds") => v... }
    # "loop_with_grouping": {
    #     "evens": [2, 4],
    #     "odds": [1, 3]
    # },

    splat_expression = ([
      {
        id: 1,
        name: "foo",
      },
      {
        id: 2,
        name: "bar",
      },
      {
        id: 3,
        name: "baz",
      },
    ])[*].name
    # "splat_expression": [
    #     "foo",
    #     "bar",
    #     "baz"
    # ],

    template_directives = <<EOT
      %{~ for v in [1, 2, 3, 4] }
        %{~ if v % 2 == 0 ~}
          ${v} is even
        %{ else ~}
          ${v*2} is doubled ${v}
        %{ endif ~}
      %{ endfor ~}
    EOT
    # "template_directives": "2 is doubled 1\n2 is even\n6 is doubled 3\n4 is even\n"
  }

  content text {
    value = <<-EOT
      arithmetic: {{ .vars.arithmetic }}

      logic: {{ .vars.logic }}

      conditionals: {{ .vars.conditionals }}

      loop_over_list: {{ .vars.loop_over_list | toJson }}

      loop_over_tuple: {{ .vars.loop_over_tuple | toJson }}

      loop_over_map: {{ .vars.loop_over_map | toJson }}

      loop_over_object: {{ .vars.loop_over_object | toJson }}

      loop_creating_object: {{ .vars.loop_creating_object | toJson }}

      loop_with_filter: {{ .vars.loop_with_filter | toJson }}

      loop_with_grouping: {{ .vars.loop_with_grouping | toJson }}

      splat_expression: {{ .vars.splat_expression | toJson }}

      template_directives: {{ .vars.template_directives }}
    EOT
  }
}
```

Evaluating this template renders the following output:

```text
arithmetic: 1 + 2 = 3

logic: true and false is false

conditionals: 2 is even

loop_over_list: [2,4,6]

loop_over_tuple: ["value is 1","value is two","value is 3"]

loop_over_map: ["key a: value 1","key b: value 2","key c: value 3"]

loop_over_object: ["key a: value 1","key b: value two","key c: value 3"]

loop_creating_object: {"1":2,"2":4,"3":6}

loop_with_filter: {"2":4,"4":8}

loop_with_grouping: {"evens":[2,4],"odds":[1,3]}

splat_expression: ["foo","bar","baz"]

template_directives:
          2 is doubled 1

          2 is even

          6 is doubled 3

          4 is even
```
