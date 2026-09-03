---
title: "`terraform_state_local` data source"
plugin:
  name: blackstork/terraform
  description: "Reads a local Terraform state file and returns its decoded JSON data"
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/terraform/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/terraform" "terraform" "v1.0.0-rc1" "terraform_state_local" "data source" >}}

## Description
Reads a local Terraform state file and returns its decoded JSON data.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `terraform_state_local` data source locally via `blackstork-cli`, you must declare the `blackstork/terraform` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/terraform" = ">= v1.0.0-rc1"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source does not accept any configuration arguments.

## Usage

This data source accepts the following arguments within a `data terraform_state_local` block:

```hcl
data terraform_state_local {
  # Required string.
  #
  # For example:
  path = "some string"
}
```