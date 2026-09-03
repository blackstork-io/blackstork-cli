---
title: "`llm_text` content provider"
plugin:
  name: blackstork/builtin
  description: "Generates text from a Go-templated prompt using the configured LLM vendor and model. Supports Google, OpenAI, Anthropic, Ollama, and xAI models"
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/builtin/"
resource:
  type: content-provider
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v1.0.0-rc1" "llm_text" "content provider" >}}

## Description
Generates text from a Go-templated prompt using the configured LLM vendor and model. Supports Google, OpenAI, Anthropic, Ollama, and xAI models.

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

