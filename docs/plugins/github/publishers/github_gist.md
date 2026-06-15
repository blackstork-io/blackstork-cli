---
title: "`github_gist` publisher"
plugin:
  name: blackstork/github
  description: "Publishes content to github gist"
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/github/"
resource:
  type: publisher
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/github" "github" "v0.4.2" "github_gist" "publisher" >}}

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `github_gist` publisher locally via `blackstork-cli`, you must declare the `blackstork/github` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/github" = ">= v0.4.2"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Supported Formats

This publisher supports the delivery of documents processed by the following formatters:

- `md`
- `html`

To specify the format, use the `format` argument inside the `publish` block to reference a specific `format` block or a formatter short name.


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
# Note: The `publish` block also accepts the generic `format` argument to link to a formatter.

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


