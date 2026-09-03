---
title: "`hub` publisher"
plugin:
  name: blackstork/builtin
  description: "Publishes the rendered document to BlackStork SaaS."
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/builtin/"
resource:
  type: publisher
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v1.0.0" "hub" "publisher" >}}

## Description
Publishes the rendered document to BlackStork SaaS.

The `hub` publisher is built into the BlackStork engine. It is available out-of-the-box and requires no installation or dependency declaration.


## Configuration

This publisher accepts the following configuration arguments within a `config publish hub` block:

```hcl
config publish hub {
  # API token
  #
  # Required string.
  # Must be non-empty
  #
  # For example:
  api_token = "some string"

  # Base URL
  #
  # Required string.
  # Must be non-empty
  #
  # For example:
  base_url = "some string"
}

```

## Usage

This publisher accepts the following arguments within a `publish hub` block:

```hcl
# The `publish` block also accepts the generic `format_ref` argument to link to a formatter.

publish hub {
}

```

