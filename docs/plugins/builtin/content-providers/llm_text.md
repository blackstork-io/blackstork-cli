---
title: "`llm_text` content provider"
plugin:
  name: blackstork/builtin
  description: ""
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/builtin/"
resource:
  type: content-provider
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v0.4.2" "llm_text" "content provider" >}}

The `llm_text` content provider is built into the BlackStork engine. It is available out-of-the-box and requires no installation or dependency declaration.

## Configuration

This content provider accepts the following configuration arguments within a `config content llm_text` block:

```hcl
config content llm_text {
  # LLM vendor name
  #
  # Required string.
  # Must be one of: "google", "openai", "anthropic", "ollama", "xai"
  #
  # For example:
  vendor = "google"

  # Model name
  #
  # Required string.
  #
  # For example:
  model = "googleai/gemini-3-flash"

  # Required string.
  #
  # For example:
  api_key = "key_value"

  # Optional string.
  # Default value:
  system_prompt = null
}
```

## Usage

This content provider accepts the following arguments within a `content llm_text` block:

```hcl
content llm_text {
  # Required string.
  #
  # For example:
  prompt = "Summarize the following text: {{.vars.text_to_summarize}}"
}
```

