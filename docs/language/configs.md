---
title: Configuration
description: Learn how to configure the BlackStork engine, declare plugin dependencies, and authenticate external integrations through data sources and content providers.
type: docs
weight: 20
---

# Configuration

## Global configuration

The `blackstork` configuration block serves as the global configuration for the engine. Use it to define global parameters, plugin dependencies, and local paths.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/elastic" = "~> 0.4.1"
    "blackstork/openai"  = "~> 0.4.1"
  }
}
```

Only one `blackstork` block can be defined per project.

{{< hint warning >}}
**BlackStork SaaS Compatibility**

The global `blackstork` configuration block is only evaluated when running templates locally via `blackstork-cli`. Within the managed BlackStork SaaS platform, plugin resolution and global configurations are built-in and cannot be modified by users.
{{< /hint >}}

### Supported arguments

- `plugin_versions`: (optional) A map of plugin dependencies. Version constraints use SemVer syntax (identical to Terraform [version constraint syntax](https://developer.hashicorp.com/terraform/language/expressions/version-constraints#version-constraint-syntax)).
- `expose_env_vars_with_pattern`: (optional) A glob pattern for environment variable names. Only environment variables matching this pattern are exposed in the evaluation context. The default value is `BLACKSTORK_*`. To expose all environment variables, set the value to `*`. See [Environment variables](#environment-variables) for details.
- `cache_dir`: (optional) Path to a local directory for storing plugins and cached data. The default value is `.blackstork` in the current working directory. If the directory does not exist, the engine will create it upon execution.

To resolve and install the dependencies defined in `plugin_versions`, execute `blackstork-cli install`.

### Supported blocks

- `plugin_registry`: (optional) Configures the plugin registry source.

  ```hcl
  plugin_registry {
    base_url = "<url>"
    mirror_dir = "<path>"
  }
  ```

  - `base_url`: (optional) The base URL of the remote plugin registry. Default: `https://registry.blackstork.io`
  - `mirror_dir`: (optional) A local filesystem path containing cached plugin binaries, useful for air-gapped environments.

### Example

```hcl
blackstork {
  cache_dir = "./.blackstork"

  plugin_registry {
    mirror_dir = "/tmp/local-mirror/plugins"
  }

  plugin_versions = {
    "blackstork/elastic" = "1.2.3"
    "blackstork/openai"  = "=11.22.33"
  }
}
```

## Block configuration

Data sources, content providers, formatters, and publishers sometimes require specific configurations (e.g., API keys, host URLs). These are defined using `config` blocks.

The `config` block must be defined at the root level of the file, outside of any `document` block. Its signature consists of the component type (`data`, `content`, `format`, or `publish`), the specific plugin name, and an optional block name.

```hcl
config <block-type> <plugin-name> "<optional-name>" {
  # ...
}
```

If `<optional-name>` is omitted, the block acts as the default configuration for that specific data source / content provider / formatter / publisher across the entire project.

If a name is provided, the configuration must be explicitly referenced inside a block that wants to use it through the `config` argument. This is required when evaluating multiple instances of the same source / provider / formatter / publisher (e.g., querying two different Splunk environments).

### Supported arguments

Configuration arguments depend on the component. See the reference for the relevant [data source]({{< ref "data-sources.md" >}}), [content provider]({{< ref "content-providers.md" >}}), [formatter]({{< ref "formatters.md" >}}), or [publisher]({{< ref "publishers.md" >}}).


### Supported blocks

- `meta`: (optional) A block containing metadata. See [Metadata](#metadata) for details.


### Example

```hcl
# Default configuration for the OpenAI content provider
config content openai_text {
  api_key       = env.OPENAI_API_KEY
  system_prompt = "You are a senior incident response analyst."
}

document "test-document" {

  data file "events_a" {
    path = "/tmp/events-a.csv"
    format = "csv"
    csv_delimiter = ";"
  }

  data file "events_b" {
    path = "/tmp/events-b.csv"
    format = "csv"
    csv_delimiter = ","
  }

  content openai_text {
    prompt = <<-EOT
      Summarize the findings:
        {{ .data.file.events_a | toPrettyJson }}
        {{ .data.file.events_b | toPrettyJson }}
    EOT
  }
}
```

## Environment variables

When executing locally via `blackstork-cli`, configurations can dynamically reference environment variables set in the executing shell or provided via a `.env` file.

{{< hint warning >}}

**BlackStork SaaS Compatibility**

Environment variables are not supported within the managed BlackStork SaaS platform. Templates relying on the `env` object or `.env` context will fail to evaluate. To ensure your templates are fully portable across both the CLI and the SaaS platform, use `input` blocks to define required runtime parameters instead.

{{< /hint >}}

For local execution, variables can be accessed directly in HCL through the global `env` object, or within Go templates and jq queries through the `.env` key in the evaluation context. Values in the process environment override values with the same name in `.env`.


```hcl
config data elasticsearch {
  basic_auth_username = "elastic"
  basic_auth_password = env.ELASTICSEARCH_PASSWORD
}

content text {
  local_var = query_jq(".env.TARGET_IPS | split(\",\") | length")
  value = "The query targeted {{ .vars.local }} IP addresses from the TARGET_IPS environment variable."
}
```

{{< hint note >}}
**CLI Context Exposure Restrictions**

To enforce the principle of least privilege, the `blackstork-cli` restricts which environment variables are exposed to templates via the evaluation context.

Only variables matching the `expose_env_vars_with_pattern` argument in the global `blackstork` block will be available under `.env` in the evaluation context. By default, this is restricted to `BLACKSTORK_*`.

This filtering does not apply to direct HCL interpolation using the `env.<VAR>` object. You are responsible for ensuring that sensitive credentials interpolated via `env` are not inadvertently leaked into rendered text blocks.
{{< /hint >}}

## Metadata

The `meta` block defines metadata for a template, document, or specific block. This is useful for cataloging community templates and tracking version history.

Supported arguments:

- `name` — String: The human-readable name of the template or block.
- `description` — String: Details regarding the block's purpose.
- `url` — String: A reference URL.
- `license` — String: The applicable license.
- `authors` — List of strings: Author names or handles.
- `tags` — List of strings: Categorization tags.
- `updated_at` — String: An ISO8601-formatted timestamp.
- `version` — String: The template or block version.

### Example

```hcl
document "mitre_ctid_campaign_report" {

  meta {
    name        = "MITRE CTID Campaign Report Template"
    description = "Highlights new information related to a threat actor or capability, focusing on changes in organizational risk."
    url         = "https://github.com/center-for-threat-informed-defense/cti-blueprints"
    license     = "Apache License 2.0"
    tags        = ["mitre", "campaign", "cti"]
    updated_at  = "2024-01-22T10:00:01+01:00"
  }

}

content text "disclaimer" {
  meta {
    name = "TLP Amber Disclaimer"
    tags = ["tlp", "compliance"]
  }
  value = "This document is TLP:AMBER."
}
```

## Next steps

See [Documents]({{< ref "documents.md" >}}) to learn how to structure data and content blocks within a document template.
