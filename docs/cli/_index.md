---
title: BlackStork CLI
description: Overview of blackstork-cli, the headless execution engine for rendering BlackStork templates locally and in CI/CD pipelines.
type: docs
weight: 60
---

# BlackStork CLI

The `blackstork-cli` is a source-available, headless execution engine for BlackStork templates. It is the exact same engine that powers the document generation within the managed BlackStork SaaS platform, packaged as a standalone binary for local environments.

When executed, the CLI parses your `.blackstork.hcl` configuration files, loads installed plugins, fetches configured data, and renders documents locally. Use `blackstork-cli install` to download declared plugin dependencies before rendering.

Because it runs entirely on your infrastructure, `blackstork-cli` is ideal for users who need to keep sensitive security data strictly on-premise, or for engineering teams integrating automated reporting directly into existing CI/CD pipelines.

## License

`blackstork-cli` is source-available and licensed under the Business Source License (BUSL) 1.1. The license permits many internal and production uses but restricts use of the software to provide a Managed Service. See the [License]({{< ref "license.md" >}}) page for the authoritative terms.

## CLI Documentation

- [Install]({{< ref "install.md" >}}) — Download and install the `blackstork-cli` binary for your operating system.
- [CLI Reference]({{< ref "cli-reference.md" >}}) — Detailed documentation on available commands, flags, and environment variables.
- [License]({{< ref "license.md" >}}) — BlackStork CLI BUSL 1.1 license.
