# BlackStork Template Authoring Guide for AI Agents

Use this guide to create syntactically correct, readable, and maintainable BlackStork template files. Treat the rules marked **must** as constraints. When a requested integration or component is not documented here, inspect the component reference under `docs/plugins/` before writing its arguments. Never invent a block type, provider name, or argument.

## Output contract

When generating a BlackStork template, an agent must:

1. Write UTF-8 text in one or more files ending in `.blackstork.hcl`.
2. Use HCL native syntax, BlackStork blocks, JQ expressions, and Go templates only in the roles described below.
3. Use only installed or declared plugins and only documented component arguments.
4. Keep secrets out of the template. Read permitted environment variables through `env.<NAME>`.
5. Run `blackstork-cli lint` against the source directory and resolve every error before returning the template.
6. Prefer the simplest structure that meets the request. Do not add speculative integrations, settings, or abstractions.

## The four language layers

BlackStork templates combine four languages. Do not mix their responsibilities.

| Layer | Syntax | Evaluated | Use it for |
|---|---|---|---|
| HCL | `name = expression`, `${expression}` | When configuration is loaded | Blocks, attributes, literals, collections, and simple expressions |
| BlackStork | `document`, `data`, `content`, `section`, and related blocks | During the evaluation pipeline | Fetching data and constructing the document tree |
| JQ | `query_jq("...")` | During data evaluation | Reading and transforming the evaluation context |
| Go templates | `{{ .vars.name }}` | When content or supported arguments render | Injecting evaluated values into text and presentation templates |

Use `query_jq()` to produce structured values. Use Go templates to interpolate values into strings. Avoid HCL template directives such as `%{ for ... }` for report logic; native `dynamic` blocks and JQ are clearer and more portable.

## HCL essentials

### Arguments and blocks

An argument assigns an expression to a name:

```hcl
value = "Static text"
```

A block has a type, may have labels, and contains arguments or nested blocks:

```hcl
content text "summary" {
  value = "Report summary"
}
```

Identifiers may contain letters, digits, underscores, and hyphens, but must not start with a digit. For maintainability, use `snake_case` identifiers and quoted labels:

```hcl
document "weekly_security_report" {
}
```

Arguments use `=`. Object entries may use `=` or `:`, but use `=` consistently. Separate list elements with commas. HCL permits omitted commas between object attributes; prefer one attribute per line without commas for multiline objects.

```hcl
vars {
  severity_order = ["critical", "high", "medium", "low"]

  owner = {
    name  = "Security Operations"
    email = "soc@example.com"
  }
}
```

Use `#` for comments. Explain intent or non-obvious constraints, not syntax. Do not leave commented-out code in generated templates.

### Strings and heredocs

Use quoted strings for short values. Escape embedded quotes with `\"` and backslashes with `\\`.

Use an indented heredoc for multiline text:

```hcl
value = <<-EOT
  First paragraph.

  Second paragraph with {{ .vars.owner.name }}.
EOT
```

The `-` in `<<-EOT` removes common leading indentation. Put the closing marker on its own line. Use a descriptive delimiter instead of `EOT` when the content itself might contain that marker.

### HCL expressions

HCL supports booleans, numbers, strings, lists, objects, conditionals, operators, `for` expressions, and splats.

```hcl
vars {
  threshold = 7
  label     = 7 >= 5 ? "high" : "low"
  doubled   = [for value in [1, 2, 3] : value * 2]
}
```

Use `${...}` only for HCL interpolation inside an HCL string. It cannot read BlackStork evaluation context paths such as `.data` or `.vars`; use `query_jq()` or a Go template for those.

## Project configuration

A project may contain one root-level `blackstork` block. Declare external plugin constraints here. Built-in components do not need plugin declarations.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/openai" = "~> 1.0"
  }

  expose_env_vars_with_pattern = "BLACKSTORK_*"
}
```

The default exposed environment-variable pattern is `BLACKSTORK_*`. Refer to an exposed value as an HCL traversal, not a string:

```hcl
api_token = env.BLACKSTORK_API_TOKEN
```

Do not use `getenv`, shell expansion such as `$NAME`, or `${NAME}` for environment variables. Do not broaden exposure to `*` unless the user explicitly requires it and accepts the security tradeoff.

## Document structure

A document is a root-level, named block and the main render target:

```hcl
document "weekly_security_report" {
  title = "Weekly Security Report"
}
```

Render it with its traversal:

```shell
blackstork-cli render document.weekly_security_report
```

Inside a document, keep blocks in this order when present:

1. `meta`
2. `input`
3. `data`
4. `vars`
5. `title`
6. `content`, `section`, and `dynamic` in intended output order
7. `format`
8. `publish`

Order matters for rendered content. Variables within a `vars` block are evaluated in declaration order, so declare a value before another variable queries it.

### Metadata

Use metadata to make reusable templates discoverable and maintainable:

```hcl
meta {
  name        = "Weekly Security Report"
  description = "Summarizes security findings for the reporting period."
  authors     = ["Security Engineering <security@example.com>"]
  version     = "1.0.0"
  updated_at  = "2026-09-01T00:00:00Z"
  tags        = ["security", "weekly"]
}
```

Use RFC 3339 timestamps and semantic versions. Do not fabricate author identities or dates; omit unknown optional fields.

## Inputs, variables, and context

### Inputs

Define runtime values with `input` blocks directly inside the document. Valid types are `bool`, `datetime`, `json`, `number`, `secret`, and `string`.

```hcl
input "reporting_period" {
  type        = "string"
  label       = "Reporting period"
  description = "Human-readable period shown in the report"
}

input "include_details" {
  type          = "bool"
  default_value = true
}
```

Add `default_value` only when a safe, meaningful default exists. A `json` input may use an HCL value as its default and may include a JSON Schema in `schema`.

CLI callers provide values with repeatable flags:

```shell
blackstork-cli --input reporting_period=2026-W35 render document.weekly_security_report
```

### Context paths

The evaluation context is JSON-like and has these principal paths:

- `.inputs.<name>`: runtime inputs.
- `.vars.<name>`: variables visible in the current scope.
- `.data.<provider>.<block_name>`: data-source results.
- `.document.meta`: document metadata.
- `.section.meta` and `.content.meta`: metadata in the corresponding local scope.
- `.deps.<block traversal>`: results explicitly requested through `depends_on`.

### Variables and JQ

Use `vars` for static values and derived structured values. A nested section or content block inherits parent variables and may shadow them locally.

```hcl
vars {
  findings = query_jq(".inputs.findings // []")
  critical_findings = query_jq(
    ".vars.findings | map(select(.severity == \"critical\"))"
  )
  critical_count = query_jq(".vars.critical_findings | length")
}
```

For multiline JQ, use a heredoc:

```hcl
summary = query_jq(<<-JQ
  .vars.findings
  | group_by(.severity)
  | map({ severity: .[0].severity, count: length })
JQ
)
```

Keep JQ transformations in `vars`, give intermediate results meaningful names, and avoid repeating a complex query across content blocks.

`local_var` is shorthand for `.vars.local` in the current block:

```hcl
content text {
  local_var = query_jq(".vars.findings | length")
  value     = "Total findings: {{ .vars.local }}"
}
```

Do not use `local_var` together with a `vars` block. Prefer a named variable when the value is important enough to reuse or explain.

## Data blocks

A data block requires a provider and a name. It may appear at the root level or directly inside a document, but not inside a section.

```hcl
data http "current_findings" {
  url                = "https://api.example.com/findings"
  format = "json"
}
```

Its result is available at `.data.http.current_findings`. Provider arguments are schema-specific: inspect `docs/plugins/<plugin>/data-sources/<provider>.md`. Do not infer an argument from another tool or provider with a similar name.

Use `config` blocks for reusable credentials or connection settings when the component documentation supports them. Keep secrets in exposed environment variables.

## Content and sections

Content blocks have a provider and an optional name inside a document or section. Root-level reusable content must be named.

```hcl
content text "executive_summary" {
  value = <<-EOT
    The report contains {{ .vars.critical_count }} critical findings.
  EOT
}
```

Sections group ordered content and establish nested variable scope:

```hcl
section "findings" {
  title       = "Findings"
  is_included = query_jq(".vars.findings | length > 0")

  content text {
    value = "Review and prioritize the findings below."
  }
}
```

Use `title = "..."` on a document or section for its heading. Use `content title` only when you need provider-specific title behavior.

Common built-in content providers include `text`, `title`, `table`, `list`, `code`, `image`, `blockquote`, and `toc`. Their arguments are not interchangeable; consult `docs/plugins/builtin/content-providers/`.

### Go templates

Go templates are valid only in arguments documented as templated strings. Access context values with paths such as `{{ .vars.name }}`. Use template functions for presentation, not data modeling.

```hcl
value = <<-EOT
  Findings: {{ .vars.findings | toPrettyJson }}
EOT
```

Keep control flow short. If a template needs deeply nested conditions or transformations, compute the value with JQ first. Quote string arguments inside template actions with escaped quotes in ordinary HCL strings; heredocs usually avoid extra HCL escaping.

### Tables and lists

```hcl
content table "finding_summary" {
  rows = query_jq(".vars.findings")

  columns = [
    { header = "ID", value = "{{ .row.value.id }}" },
    { header = "Severity", value = "{{ .row.value.severity }}" },
    { header = "Title", value = "{{ .row.value.title }}" }
  ]
}

content list "recommendations" {
  items         = query_jq(".vars.recommendations")
  format        = "unordered"
  item_template = "{{ . }}"
}
```

Within a table column template, the current row is `.row.value`, its zero-based index is `.row.index`, and the column index is `.col.index`.

## Conditional and repeated content

Use `is_included` for one conditional block:

```hcl
content text {
  is_included = query_jq(".inputs.include_details")
  value       = "Detailed analysis goes here."
}
```

Use `dynamic` to repeat content or sections for a list. Each iteration exposes `.vars.dynamic_item` and `.vars.dynamic_item_index`.

```hcl
dynamic {
  items = query_jq(".vars.findings")

  section "finding" {
    title = "{{ .vars.dynamic_item.id }}: {{ .vars.dynamic_item.title }}"

    content text {
      value = "Severity: **{{ .vars.dynamic_item.severity }}**"
    }
  }
}
```

A `dynamic` block may contain `content`, `section`, or nested `dynamic` blocks. Use a dynamic block instead of HCL `%{ for }` directives when producing document structure.

## Dependencies

Blocks normally follow declaration and structural order. When one content block needs another block's evaluated result, name the dependency and declare it with its quoted traversal:

```hcl
content code "raw_payload" {
  language = "json"
  value    = "{\"status\":\"ok\"}"
}

content code "payload_copy" {
  depends_on = ["content.code.raw_payload"]
  language   = "json"
  value      = "{{ .deps.content.code.raw_payload | toPrettyJson }}"
}
```

Use `depends_on` only for real evaluation dependencies. Do not use it merely to express display order.

## Reusable blocks and references

Define reusable `data`, `content`, `section`, `format`, or `publish` blocks at the root level with names. Include or override one with a `ref` block and an unquoted `base` traversal:

```hcl
content text "standard_disclaimer" {
  value = "This report is confidential."
}

document "example" {
  content ref "report_disclaimer" {
    base = content.text.standard_disclaimer
  }
}
```

The traversal components match the referenced block signature. Examples include `data.http.findings`, `content.text.standard_disclaimer`, and `section.executive_summary`.

An anonymous reference inherits the base block's name. Give references explicit unique names when the same base is used more than once in a scope. Put overrides directly in the `ref` block. Avoid copying a reusable block merely to change one argument.

## Formatting and publishing

Format blocks are root-level or direct children of a document. A formatter name is required; a name is optional inside a document.

```hcl
format md "standard_markdown" {
}
```

Publish blocks deliver already formatted output. Link a publisher to a formatter with an unquoted `format_ref` traversal:

```hcl
document "example" {
  format md "report_markdown" {
  }

  publish local_file "markdown_file" {
    format_ref = document.format.md.report_markdown
    path       = "output/example.md"
  }
}
```

Run publishers only when explicitly requested:

```shell
blackstork-cli render document.example --publish
```

Publisher and formatter arguments are component-specific. Inspect `docs/plugins/*/formatters/` and `docs/plugins/*/publishers/` before generating them. Treat output paths and remote publishing as side effects; do not execute them during validation without authorization.

## Canonical complete template

The following template uses only built-in, offline-safe components and is a good starting structure:

```hcl
document "security_findings_report" {
  meta {
    name        = "Security Findings Report"
    description = "Summarizes supplied security findings."
    version     = "1.0.0"
    tags        = ["security", "findings"]
  }

  input "reporting_period" {
    type        = "string"
    label       = "Reporting period"
    description = "Period covered by this report"
  }

  input "findings" {
    type          = "json"
    default_value = []
    schema = {
      type = "array"
      items = {
        type = "object"
        properties = {
          id       = { type = "string" }
          title    = { type = "string" }
          severity = { type = "string" }
        }
        required = ["id", "title", "severity"]
      }
    }
  }

  vars {
    findings      = query_jq(".inputs.findings")
    finding_count = query_jq(".vars.findings | length")
  }

  title = "Security Findings Report"

  content text "overview" {
    value = <<-EOT
      **Reporting period:** {{ .inputs.reporting_period }}

      **Total findings:** {{ .vars.finding_count }}
    EOT
  }

  section "findings" {
    title       = "Findings"
    is_included = query_jq(".vars.finding_count > 0")

    content table "finding_table" {
      rows = query_jq(".vars.findings")
      columns = [
        { header = "ID", value = "{{ .row.value.id }}" },
        { header = "Severity", value = "{{ .row.value.severity }}" },
        { header = "Title", value = "{{ .row.value.title }}" }
      ]
    }
  }

  section "no_findings" {
    is_included = query_jq(".vars.finding_count == 0")

    content text {
      value = "No findings were supplied for this reporting period."
    }
  }

  format md "report_markdown" {
  }
}
```

## Style and maintainability rules

- Use one responsibility per file when a project grows: configuration, reusable components, document definitions, and format definitions can live in separate `.blackstork.hcl` files in the same source directory.
- Use stable, descriptive `snake_case` names. Name blocks that are referenced, dependencies, or operationally important; anonymous blocks are fine for simple one-off content.
- Keep the document's block order aligned with the rendered narrative.
- Extract repeated or complex JQ into clearly named variables.
- Prefer inputs over hard-coded values that change between runs.
- Prefer references over copied reusable blocks.
- Keep presentation in `format` blocks and data transformation in `vars`/JQ.
- Avoid excessive nesting. Create a section only when it adds document hierarchy, variable scope, conditional grouping, or reuse.
- Use `is_included` instead of embedding large Go-template conditionals in text.
- Never hard-code credentials, access tokens, private keys, or environment-specific absolute paths.
- Do not add empty blocks except when a default formatter instance is intentionally selected.
- Align adjacent `=` signs as produced by `hclfmt` when available, but do not hand-align distant or unrelated attributes.

## Agent validation checklist

Before returning generated files, verify all of the following:

- Every file ends in `.blackstork.hcl` and is UTF-8.
- Every document and every root-level reusable block has a unique name.
- Every `base` and `format_ref` is an unquoted traversal and resolves to a declared block.
- Every `depends_on` entry is a quoted traversal and has a uniquely named target.
- Each `.data.<provider>.<name>`, `.inputs.<name>`, and `.vars.<name>` path exists in scope.
- Variables are declared before later variables query them.
- Every provider argument exists in the relevant generated plugin documentation.
- Secrets use permitted `env.BLACKSTORK_*` variables and are not embedded as literals.
- Multiline strings use correctly paired heredoc delimiters.
- Go-template delimiters and JQ string quoting are balanced.
- Side-effecting publish blocks are intentional and linked to compatible formats.
- Comments explain intent and no dead code remains.

Run structural lint from the project root:

```shell
blackstork-cli --source-dir . lint
```

If installed plugin bodies should also be checked, run:

```shell
blackstork-cli --source-dir . lint --full
```

Finally, render offline-safe templates without `--publish` and inspect the output:

```shell
blackstork-cli --source-dir . \
  --input reporting_period=2026-W35 \
  render document.security_findings_report
```

Do not claim a template is valid merely because its HCL parses. BlackStork lint must also resolve block signatures, references, traversals, and component schemas.
