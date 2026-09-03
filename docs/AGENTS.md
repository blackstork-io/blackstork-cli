# Documentation maintenance

These instructions apply to `docs/` and supplement the repository `AGENTS.md`.

- Write for template authors and CLI users. Lead with the task or outcome.
- Use current BlackStork terminology and `.blackstork.hcl` syntax.
- Use `BLACKSTORK_` for environment-variable examples and never include real
  credentials.
- Verify command examples against `blackstork-cli <command> --help`.
- Verify runner examples against their current schemas under `plugins/`.
- Keep code examples minimal but executable; do not use pseudocode in an HCL
  code fence unless it is clearly labeled.
- Follow the repository Markdown lint and Vale configuration.

Files under `docs/plugins/` are generated. Change the runner schema or a
template under `tools/docgen/`, then run `make generate-docs`. Other pages under
`docs/` are maintained directly.
