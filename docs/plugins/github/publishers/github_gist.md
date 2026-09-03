---
title: "`github_gist` publisher"
plugin:
  name: blackstork/github
  description: "Creates a GitHub Gist from the rendered document or updates an existing Gist when `gist_id` is provided."
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/github/"
resource:
  type: publisher
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/github" "github" "v1.0.0-rc1" "github_gist" "publisher" >}}

## Description
Creates a GitHub Gist from the rendered document or updates an existing Gist when `gist_id` is provided.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `github_gist` publisher locally via `blackstork-cli`, you must declare the `blackstork/github` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/github" = ">= v1.0.0-rc1"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.


## Configuration

This publisher accepts the following configuration arguments within a `config publish github_gist` block:

```hcl
config publish github_gist {
  # Required string.
  #
  # For example:
  github_token = "some string"
}

```

## Usage

This publisher accepts the following arguments within a `publish github_gist` block:

```hcl
# The `publish` block also accepts the generic `format_ref` argument to link to a formatter.

publish github_gist {
  # Optional string.
  # Default value:
  description = null

  # Optional string.
  # Default value:
  filename = null

  # Optional bool.
  # Default value:
  make_public = false

  # Optional string.
  # Default value:
  gist_id = null
}

```

