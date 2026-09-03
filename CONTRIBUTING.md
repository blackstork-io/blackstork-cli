# Contributing to blackstork-cli

Thank you for improving BlackStork. Contributions can include bug fixes,
plugins, tests, documentation, and focused design proposals.

## Before you start

- Read [AGENTS.md](AGENTS.md) for repository conventions.
- Read [ARCHITECTURE.md](ARCHITECTURE.md) for package boundaries and execution
  flow.
- Search existing issues before opening a duplicate.
- For a security vulnerability, follow [SECURITY.md](SECURITY.md) instead of
  opening a public issue.
- Review [LICENSE](LICENSE). Contributions to this repository are distributed
  under the project's Business Source License terms.

## Development setup

Install the Go version declared in `go.mod`, clone the repository, and download
dependencies:

```bash
git clone https://github.com/blackstork-io/blackstork-cli.git
cd blackstork-cli
go mod download
go build ./...
```

Some integration tests use Docker containers. SQLite tests require CGO and a C
toolchain. Generation scripts install or invoke additional pinned tools as
needed.

## Make a change

1. Create a branch from the current `main` branch.
2. Keep the change focused and preserve unrelated work in the checkout.
3. Add or update tests for behavioral changes.
4. Update public documentation when behavior, syntax, commands, or runner
   schemas change.
5. Regenerate derived files when their sources change.
6. Run the relevant validation commands.

### Go changes

Format changed Go files with `gofmt`. The repository-wide formatter also runs
`gofumpt`, `gci`, and `go mod tidy`:

```bash
make format
```

Because this command can modify files beyond the one being edited, review the
resulting diff carefully.

### Plugin changes

Runner schemas are public documentation. Describe each runner and attribute
for a template author, mark secrets correctly, and keep examples safe. After a
schema change, regenerate the plugin reference:

```bash
make generate-docs
```

Do not edit generated pages under `docs/plugins/` directly.

### Generated code

When protocol or generation inputs change, run:

```bash
make generate
```

Commit generated outputs with their source changes.

## Validate

Start with the narrowest relevant checks:

```bash
go test ./path/to/changed/package
go build ./...
git diff --check
```

Before submitting a substantial change, run the repository checks when the
local environment supports them:

```bash
make test
make lint
```

Integration tests can require Docker, network access, CGO, or test credentials.
If a check cannot run, state that clearly in the pull request. Do not hide a
failure or weaken a test to make an unrelated change pass.

## Documentation style

- Use direct, task-oriented language.
- Explain what a user can do before describing implementation details.
- Use sentence case for headings.
- Keep terminology consistent with CLI help and runner schemas.
- Test commands and template examples instead of relying on memory.

## Commits and pull requests

Use a Conventional Commit subject:

```text
fix(parser): preserve metadata across references
feat(opencti): add relationship filters
docs(cli): clarify data target syntax
```

A pull request should explain the problem, the chosen behavior, important
tradeoffs, and the validation performed. Keep mechanical generation changes in
the same commit as the source change that requires them.

By submitting a contribution, you confirm that you have the right to provide
it under the repository's license terms.
