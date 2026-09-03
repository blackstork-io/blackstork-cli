# Parser development

These instructions apply to `parser/` and supplement the repository
`AGENTS.md`.

- Treat parsed HCL files and `hclsyntax` nodes as shared, immutable source data.
  A document can reference a standalone block that is parsed again later.
- Do not remove common blocks or attributes from a shared body. Build filtered
  views or copies for decoding instead.
- Parse common metadata consistently for every block kind that supports
  `meta`.
- Preserve source ranges so diagnostics can point to the original file and
  expression.
- Keep discovery recursive and deterministic. New file formats or compatibility
  behavior require explicit tests and documentation.
- Add regression tests for repeated parsing, reference order, metadata
  preservation, invalid nesting, and duplicate definitions as applicable.

Run `go test ./parser ./eval ./engine` after parser changes, followed by
`go build ./...`.
