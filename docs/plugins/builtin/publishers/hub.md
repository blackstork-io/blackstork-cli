---
title: "`hub` publisher"
plugin:
  name: blackstork/builtin
  description: "Publish documents to BlackStork cloud"
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/builtin/"
resource:
  type: publisher
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v0.4.2" "hub" "publisher" >}}

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
# Note: The `publish` block also accepts the generic `format` argument to link to a formatter.

publish hub {
}

```


