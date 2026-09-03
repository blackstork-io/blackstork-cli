# Working on blackstork-cli

This file is the canonical repository guide for coding agents. It applies to
the entire repository. A nested `AGENTS.md` adds instructions for its subtree.

## Project

`blackstork-cli` is the source-available Go execution engine for BlackStork
templates. It parses `.blackstork.hcl` files, resolves plugins, evaluates data
and content blocks, formats documents, and publishes the results.

Read [ARCHITECTURE.md](ARCHITECTURE.md) before changing package boundaries or
execution flow. Read
[BLACKSTORK_TEMPLATE_AUTHORING.md](BLACKSTORK_TEMPLATE_AUTHORING.md) before
creating or editing BlackStork templates.

## Repository map

- `cmd/`: CLI commands and user-facing command help.
- `engine/`: orchestration of parsing, plugin loading, evaluation, formatting,
  and publishing.
- `parser/`: HCL discovery and parsing into BlackStork definitions.
- `eval/`: runtime evaluation of parsed documents and plugin actions.
- `specs/`: block definitions, schemas, constraints, and validators.
- `plugin/`: plugin contracts, content types, data representation, protocol,
  resolution, and test helpers.
- `plugins/`: built-in and separately distributed plugin runners.
- `proto/`: protocol source definitions.
- `codegen/`: formatting, generation, and documentation scripts.
- `tools/`: repository tooling, including plugin metadata and doc generation.
- `docs/`: product documentation and generated plugin reference pages.

## Environment

- Use the Go version declared in `go.mod`.
- Run `go mod download` after cloning if dependencies are not already cached.
- Some integration tests require Docker and CGO.
- The CLI reads a local `.env` file. Never commit `.env` or credentials.
- Environment variables exposed to templates use the `BLACKSTORK_` prefix by
  default.

## Common commands

```bash
go build ./...
make test
make lint
make generate
make generate-docs
```

Prefer focused package tests while iterating, then run the broadest relevant
validation before handing off a change.

`make lint` runs `make format` first. Formatting executes `go mod tidy`,
`gofumpt`, and `gci`, so it can modify files. Inspect the diff afterward and do
not discard unrelated user changes.

## Change rules

- Preserve existing behavior unless the request requires a behavioral change.
- Keep changes focused; do not reformat or rewrite unrelated files.
- Do not mutate shared HCL syntax trees during parsing. Multiple definitions
  can reference the same source node.
- Preserve common blocks such as `meta` when parsing or resolving references.
- Treat CLI help, runner `Doc` fields, examples, and diagnostics as public API.
- Use current BlackStork terminology in user-facing text and APIs.
- Use `BLACKSTORK_` for documented environment-variable examples.
- Add or update tests for behavior changes and regression fixes.
- Report pre-existing or environment-dependent validation failures separately.

## Generated files

- Change plugin runner schemas or `tools/docgen/*.gotempl`, then run
  `make generate-docs`. Do not hand-edit `docs/plugins/` pages.
- Change protocol or generation inputs, then run `make generate` and commit the
  resulting generated files.
- Do not edit generated `*.pb.go`, generated mocks, or generated templ files by
  hand.

## Plugin changes

Follow [plugins/AGENTS.md](plugins/AGENTS.md). Every runner description must be
written for template authors and must agree with its current configuration and
argument schema.

## Parser changes

Follow [parser/AGENTS.md](parser/AGENTS.md). Parser changes require focused
regression tests for shared-AST behavior, references, and metadata preservation
when those areas are affected.

## Documentation changes

Follow [docs/AGENTS.md](docs/AGENTS.md). Keep examples executable and use the
current `.blackstork.hcl` syntax.

## Completion checklist

Before handing off a change:

1. Review the diff for unrelated edits and secrets.
2. Run `gofmt` on changed Go files.
3. Run focused tests for changed packages.
4. Run `go build ./...` for cross-package compile validation.
5. Run the relevant generation command and verify generated files are current.
6. Run `git diff --check`.
7. State which checks passed and identify any checks that could not run.

## Commits

Use Conventional Commit subjects, such as `fix(parser): preserve section
metadata` or `docs(plugins): clarify runner descriptions`. Keep generated
documentation in the same commit as the schema change that produced it.

## License

This project uses the Business Source License 1.1 with an Additional Use Grant
and a future Apache License 2.0 change license. See [LICENSE](LICENSE). Do not
describe the project as open source before the applicable change date.
