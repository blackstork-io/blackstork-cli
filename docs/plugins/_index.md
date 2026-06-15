---
title: Plugins
description: Overview of the BlackStork plugin ecosystem. Learn how to declare dependencies, install integrations, and extend the engine's capabilities.
type: docs
weight: 50
---

# Plugins

Plugins extend the core capabilities of the BlackStork engine. They are packaged integrations that allow your templates to interact with external APIs, create specific types of content and output different file formats.

A single plugin typically provides a combination of the following components:

Fabric plugins implement various [data sources]({{< ref "data-sources.md" >}}), [content providers]({{< ref "content-providers.md" >}}) and [publishers]({{< ref "publishers.md" >}}):

- **[Data sources]({{< ref "data-sources.md" >}})**: integrations to query structured data from external security tools, databases, or local files.
- **[Content providers]({{< ref "content-providers.md" >}})**: renderers that generate text, tables, and charts locally, or via external APIs (such as LLMs).
- **[Formatters]({{< ref "formatters.md" >}})**: specifications that compile the evaluated document into final file formats (Markdown, HTML, PDF).
- **[Publishers]({{< ref "publishers.md" >}})**: outgoing integrations that route and save rendered documents to external destinations.

## Dependencies

To use a plugin's components in your templates, you must declare the plugin as a dependency in your global configuration block (see [Global configuration]({{< ref "../language/configs.md/#global-configuration" >}})).
A plugin identifier consists of a namespace (the author or vendor) and a short name. 

For example, the [`blackstork/elastic`]({{< ref "./elastic/" >}}) plugin provides the [Elasticsearch data source]({{< ref "./elastic/data-sources/elasticsearch" >}}).

Specify the required plugins and their version constraints within the `blackstork` block at the root of your configuration:

```hcl
blackstork {
  plugin_versions = {
    "blackstork/elastic" = ">= 0.4.0"
    "blackstork/openai"  = "~> 0.3.1"
  }
}
```

## Installation

Plugin releases are maintained independently from the core BlackStork engine and follow their own versioning lifecycles.

If you are using the BlackStork SaaS platform, plugin dependencies are resolved automatically by the web engine.

If you are executing builds locally or in a CI/CD pipeline, use the CLI to resolve and download your
declared dependencies. Running the install command fetches the required plugins from the registry
and places them in your local `.blackstork/` directory:

```bash
blackstork-cli install
```

## Available plugins

{{< plugins >}}
