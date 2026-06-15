---
title: "`openai_text` content provider"
plugin:
  name: blackstork/openai
  description: ""
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/openai/"
resource:
  type: content-provider
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/openai" "openai" "v0.4.2" "openai_text" "content provider" >}}

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `openai_text` content provider locally via `blackstork-cli`, you must declare the `blackstork/openai` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/openai" = ">= v0.4.2"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This content provider accepts the following configuration arguments within a `config content openai_text` block:

```hcl
config content openai_text {
  # Optional string.
  # Default value:
  system_prompt = null

  # Required string.
  #
  # For example:
  api_key = "some string"

  # Optional string.
  # Default value:
  organization_id = null
}
```

## Usage

This content provider accepts the following arguments within a `content openai_text` block:

```hcl
content openai_text {
  # Required string.
  #
  # For example:
  prompt = "Summarize the following text: {{.vars.text_to_summarize}}"

  # Optional string.
  # Must be non-empty
  # Default value:
  model = "gpt-3.5-turbo"
}
```

