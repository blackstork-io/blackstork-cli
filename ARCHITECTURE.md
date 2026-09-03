# blackstork-cli architecture

`blackstork-cli` is a headless document-generation engine. It turns
`.blackstork.hcl` templates and structured data into document content, applies
a formatter, and optionally sends the result to a publisher.

## Execution pipeline

```text
.blackstork.hcl files
        │
        ▼
  parser registry ────── global configuration and block definitions
        │
        ▼
 plugin resolution ───── installed external runners + built-in runners
        │
        ▼
 document evaluation ─── inputs → data → variables → content and sections
        │
        ▼
     formatter ────────── Markdown, HTML, or plugin-provided output
        │
        ▼
     publisher ────────── stdout, local file, BlackStork SaaS, or plugin target
```

The pipeline separates structured data, document structure, presentation, and
delivery. A template can therefore reuse the same evaluated content with
different formatters or publishers.

## Main packages

### `cmd`

Defines the Cobra CLI. Commands create an `engine.Engine`, load the source
directory, and invoke a bounded workflow:

- `install` resolves and installs required plugins;
- `lint` parses and validates templates;
- `data` evaluates selected data blocks; and
- `render` evaluates, formats, and either prints or publishes a document.

CLI descriptions, flag help, examples, and diagnostics are user-facing API.

### `engine`

Coordinates the end-to-end workflow. The engine owns the parsed block registry,
source file map, global configuration, plugin resolver, runner registry, and
default formatter and publisher actions.

It is responsible for ordering phases and collecting diagnostics. Parsing,
runtime evaluation, and provider-specific behavior remain in their respective
packages.

### `parser`

Discovers template files and parses HCL syntax into definitions. The registry
stores documents and reusable data, content, section, format, and publish
blocks.

HCL syntax nodes can be shared by several definitions and parsing passes. Code
in this package must treat the source AST as immutable. Removing a common block
such as `meta` while parsing one reference can corrupt a later parse of the
same standalone block.

### `specs`

Defines BlackStork block structures and the schema machinery used to decode
runner configuration and arguments. Constraints validate required values,
types, ranges, and allowed values.

### `eval`

Loads parsed definitions into runtime actions and evaluates them against the
document data context. It handles inputs, variables, dependencies, dynamic
blocks, sections, content selection, and calls to plugin runners.

The runtime context is JSON-like and represented with `plugindata` values.

### `plugin`

Defines the contracts for four runner roles:

- data sources retrieve structured data;
- content providers create document elements;
- formatters serialize evaluated content; and
- publishers deliver formatted documents.

Built-in runners execute in process. External plugins use the versioned plugin
API under `plugin/pluginapi` and are resolved through the plugin registry and
lock file.

### `plugins`

Contains the built-in runners and official external integrations. External
service clients are kept behind loader interfaces so behavior can be tested
without live credentials.

Runner schemas are also documentation sources. `tools/docgen` turns their
`Doc`, `Config`, and `Args` fields into the pages under `docs/plugins/`.

## Template evaluation context

A document normally uses these context namespaces:

- `.inputs`: runtime values declared by document `input` blocks;
- `.data`: results returned by data sources;
- `.vars`: static or derived values in the current scope;
- `.env`: environment variables allowed by the global exposure pattern;
- `.deps`: explicitly evaluated block dependencies; and
- block-specific metadata and iteration values.

HCL defines the configuration structure. JQ transforms structured runtime data.
Go templates interpolate evaluated values into supported string attributes.
See [BLACKSTORK_TEMPLATE_AUTHORING.md](BLACKSTORK_TEMPLATE_AUTHORING.md).

## Formatting and publishing

Evaluation first produces a formatter-independent content tree. A requested or
document-defined formatter converts that tree to a serialized representation.
Without `--publish`, the CLI uses the standard-output publisher. With
`--publish`, it executes the publishers defined by the document, each with its
referenced formatter.

## Generated artifacts

- Protocol and mock outputs are generated from their source definitions with
  `make generate`.
- Plugin reference pages are generated from runner schemas and doc templates
  with `make generate-docs`.
- Release artifacts are built with GoReleaser configuration in the repository
  root.

Generated artifacts should change only with the source that produces them.

## Security boundaries

- Template environment access is restricted by an exposure pattern whose
  default prefix is `BLACKSTORK_`.
- Runner schema fields containing credentials must be marked secret.
- Diagnostics and logs must not include tokens or secret values.
- A template or remote data source is untrusted input; parsing and rendering
  must not grant it unintended filesystem, process, or network access.
- Publishers are explicit delivery boundaries and should not run unless the
  selected workflow requests them.
