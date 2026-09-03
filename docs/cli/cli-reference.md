---
title: CLI Reference
description: Command-line reference for blackstork-cli. Learn how to install plugins, test data queries, and render documents locally.
type: docs
weight: 20
---

# CLI Reference

The `blackstork-cli` binary is the command-line interface for the BlackStork execution engine. It allows you to parse configurations, resolve plugins, evaluate data queries, and render final documents directly from your terminal.

See the [license details]({{< ref "license.md" >}}) before integrating the CLI into a product or service.

## Core commands

The CLI provides four subcommands for working with templates:

- `install` — Resolves, downloads, and caches plugin dependencies declared in the global `blackstork` block. Use `--upgrade` to upgrade within configured version constraints.
- `lint` — Checks all templates for syntax and structural errors without executing plugins. Use `--full` to validate plugin block bodies against installed runner schemas.
- `data TARGET` — Executes a standalone target in the form `data.<source>.<name>` or document data blocks matching `document.<document>.data[.<source>[.<name>]]`, then writes formatted JSON to standard output.
- `render TARGET` — Renders a target in the form `document.<name>`. It writes Markdown to standard output by default or executes the document's publishers when given `--publish`.

To view the full list of available commands and global flags, run `blackstork-cli --help`:

```text
$ blackstork-cli --help
Usage:
  blackstork-cli [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  data        Execute data blocks and print their results
  help        Help about any command
  install     Install required plugins
  lint        Validate BlackStork templates
  render      Render a document

Flags:
      --debug               enable debug mode and write telemetry to .blackstork/debug
  -h, --help                help for blackstork-cli
      --log-format string   log output format: plain or json (default "plain")
  -i, --input stringArray   set a document input as <name>=<value>; repeat for multiple inputs
      --log-level string    minimum logging level: debug, info, warn, error (default "warn")
      --source-dir string   directory to scan recursively for BlackStork template files (default ".")
  -v, --verbose             enable debug-level logging
      --version             version for blackstork-cli

Use "blackstork-cli [command] --help" for more information about a command.
```

## Source directory

When executed, `blackstork-cli` recursively searches the target directory for `.blackstork.hcl` files, merges their definitions, and parses the configuration.

By default, the CLI targets the current working directory (`.`). You can instruct the engine to load files from a different location by using the global `--source-dir` flag:

```bash
$ blackstork-cli render document.executive_summary --source-dir /path/to/templates/
```

## Pass document inputs

Pass an input as `--input name=value` or `-i name=value`. Repeat the flag to pass multiple values:

```bash
blackstork-cli render document.executive_summary \
  --input environment=production \
  --input 'findings=[{"id":"CVE-2026-1234"}]'
```

Input definitions support `bool`, `datetime`, `json`, `number`, `secret`, and `string`. If the command does not provide a value, the CLI uses `default_value`; without a default, it prompts on standard input. Supply every required input explicitly in non-interactive environments.

## Render options

- `--format` selects a formatter traversal, such as `format.html`, `format.html.report`, or `document.format.html.report`.
- `--only-with-tags` renders content whose metadata contains every comma-separated requested tag.
- `--replace-data-with` loads the document data layer from a JSON object instead of executing data blocks.
- `--publish` executes every publisher defined in the document instead of the default standard-output publisher.

## Next steps

See the [Tutorial]({{< ref "tutorial.md" >}}) for a practical guide on using these commands to build and evaluate your first report template.
