---
title: "`azure_openai_text` content provider"
plugin:
  name: blackstork/microsoft
  description: ""
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/fabric/tree/main/plugins/microsoft/"
resource:
  type: content-provider
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/microsoft" "microsoft" "v0.4.2" "azure_openai_text" "content provider" >}}

## Installation

To use `azure_openai_text` content provider, you must install the plugin `blackstork/microsoft`.

To install the plugin, add the full plugin name to the `plugin_versions` map in the Fabric global configuration block (see [Global configuration]({{< ref "configs.md#global-configuration" >}}) for more details), as shown below:

```hcl
fabric {
  plugin_versions = {
    "blackstork/microsoft" = ">= v0.4.2"
  }
}
```

Note the version constraint set for the plugin.


#### Configuration

The content provider supports the following configuration arguments:

```hcl
config content azure_openai_text {
  # Required string.
  #
  # For example:
  api_key = "some string"

  # Required string.
  #
  # For example:
  resource_endpoint = "some string"

  # Required string.
  #
  # For example:
  deployment_name = "some string"

  # Optional string.
  # Default value:
  api_version = "2024-02-01"
}
```

#### Usage

The content provider supports the following execution arguments:

```hcl
content azure_openai_text {
  # Required string.
  #
  # For example:
  prompt = "Summarize the following text: {{.vars.text_to_summarize}}"

  # Optional number.
  # Default value:
  max_tokens = 1000

  # Optional number.
  # Default value:
  temperature = 0

  # Optional number.
  # Default value:
  top_p = null

  # Optional number.
  # Default value:
  completions_count = 1
}
```

