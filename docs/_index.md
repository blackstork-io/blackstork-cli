---
title: Introduction
description: Technical documentation for the BlackStork reporting engine. Learn how to write templates, configure data integrations, and execute builds locally.
menuName: Docs
type: docs
RightToC: false
weight: 1
---

# BlackStork Documentation

BlackStork is an engine for automating cybersecurity and compliance reporting. This documentation covers the BlackStork configuration language used to build templates, the plugins with the integrations, and `blackstork-cli`, the source-available command-line tool for local execution.

The BlackStork configuration language (BCL) provides a declarative syntax for defining data requirements, document layout and document format inside the template. Templates are modular, meaning integration configs, data mutations and content blocks can be reused across different reports.

When a template is evaluated, the BlackStork engine executes a deterministic pipeline: it employes specified plugins to fetch data from external sources, and processes that data through your defined logic and content blocks, and renders a standardized document using your chosen formatter.

Because the templating language is decoupled from the execution environment, your configurations are fully portable. They can be built and evaluated collaboratively within the BlackStork SaaS platform, or executed locally and within CI/CD pipelines using `blackstork-cli`.

## Building Templates

- [Tutorial]({{< ref "tutorial.md" >}}) — A step-by-step guide to writing your first template and rendering a document.
- [Language]({{< ref "language.md" >}}) — Syntax and core concepts of the BlackStork configuration language.

## Plugins & Integrations

- [Plugins]({{< ref "plugins/_index.md" >}}) — Overview of the plugin registry and how to extend the engine's capabilities.
- [Data Sources]({{< ref "plugins/data-sources.md" >}}) — Supported integrations for querying structured data (e.g., SIEMs, TIPs, APIs).
- [Content Providers]({{< ref "plugins/content-providers.md" >}}) — Supported integrations for content generation.
- [Formatters]({{< ref "plugins/formatters.md" >}}) — Supported output formats and styling options for rendered documents.
- [Publishers]({{< ref "plugins/publishers.md" >}}) — Outgoing integrations for routing and storing rendered documents.

## Local Execution with `blackstork-cli`

- [Install]({{< ref "install.md" >}}) — Instructions for downloading and installing `blackstork-cli` binary.
- [CLI Reference]({{< ref "cli/cli-reference.md" >}}) — Usage details for `blackstork-cli`.
