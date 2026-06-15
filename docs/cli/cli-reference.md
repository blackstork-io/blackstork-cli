---
title: CLI Reference
description: Command-line reference for blackstork-cli. Learn how to install plugins, test data queries, and render documents locally.
type: docs
weight: 20
---

# CLI Reference

The `blackstork-cli` binary is the command-line interface for the BlackStork execution engine. It allows you to parse configurations, resolve plugins, evaluate data queries, and render final documents directly from your terminal.

{{< hint warning >}}
**License Restrictions**

`blackstork-cli` is licensed under the Business Source License (BUSL) 1.1. It is free for internal, academic, and non-commercial use, but cannot be embedded into competing commercial software or offered as a managed service. See the [License details]({{< ref "license.md" >}}) before integrating the CLI into your infrastructure.
{{< /hint >}}

## Core commands

The CLI provides three primary sub-commands for interacting with your templates:

- `install` - Downloads and caches all plugin dependencies declared in your global `blackstork` configuration block. See [Installing plugins]({{< ref "install.md#installing-plugins" >}}) for details.
- `data` - Executes a single specified `data` block and outputs the resulting structured JSON to standard output. This is highly useful for testing and debugging API queries before rendering a full document.
- `render` - Evaluates a specified `document` block, executing all nested data, content, format, and publish blocks. By default, it outputs the rendered Markdown to standard output unless the `--publish` flag is provided.

To view the full list of available commands and global flags, run `blackstork-cli --help`:

```text
$ blackstork-cli --help
Usage:
  blackstork-cli [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  data        Execute a single data block
  help        Help about any command
  install     Install plugins
  render      Render the document

Flags:
      --color               enables colorizing the logs and diagnostics (if supported by the terminal and log format) (default true)
  -h, --help                help for blackstork-cli
      --log-format string   format of the logs (plain or json) (default "plain")
      --log-level string    logging level ('debug', 'info', 'warn', 'error') (default "info")
      --source-dir string   a path to a directory with *.blackstork.hcl files (default ".")
  -v, --verbose             a shortcut to --log-level debug
      --version             version for blackstork-cli

Use "blackstork-cli [command] --help" for more information about a command.
```

## Source directory

When executed, `blackstork-cli` searches the target directory for all files with the `.blackstork.hcl` extension, merges them into a single evaluation context, and parses the configurations.

By default, the CLI targets the current working directory (`.`). You can instruct the engine to load files from a different location by using the global `--source-dir` flag:

```bash
$ blackstork-cli render document.executive_summary --source-dir /path/to/templates/
```

## Next steps

See the [Tutorial]({{< ref "tutorial.md" >}}) for a practical guide on using these commands to build and evaluate your first report template.
