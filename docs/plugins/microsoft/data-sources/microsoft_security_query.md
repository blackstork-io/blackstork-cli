---
title: "`microsoft_security_query` data source"
plugin:
  name: blackstork/microsoft
  description: "Runs a Microsoft Defender XDR advanced hunting query and returns the query results"
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/microsoft/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/microsoft" "microsoft" "v1.0.0-rc1" "microsoft_security_query" "data source" >}}

## Description
Runs a Microsoft Defender XDR advanced hunting query and returns the query results.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `microsoft_security_query` data source locally via `blackstork-cli`, you must declare the `blackstork/microsoft` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/microsoft" = ">= v1.0.0-rc1"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data microsoft_security_query` block:

```hcl
config data microsoft_security_query {
  # The Azure client ID
  #
  # Required string.
  #
  # For example:
  client_id = "some string"

  # The Azure client secret. Required if `private_key_file` or `private_key` is not provided.
  #
  # Optional string.
  # Default value:
  client_secret = null

  # The Azure tenant ID
  #
  # Required string.
  #
  # For example:
  tenant_id = "some string"

  # The path to the private key file. Ignored if `private_key` or `client_secret` is provided.
  #
  # Optional string.
  # Default value:
  private_key_file = null

  # The private key contents. Ignored if `client_secret` is provided.
  #
  # Optional string.
  # Default value:
  private_key = null

  # The key passphrase. Ignored if `client_secret` is provided.
  #
  # Optional string.
  # Default value:
  key_passphrase = null
}
```

## Usage

This data source accepts the following arguments within a `data microsoft_security_query` block:

```hcl
data microsoft_security_query {
  # Advanced hunting query to run
  #
  # Required string.
  #
  # For example:
  query = "DeviceRegistryEvents | where Timestamp >= ago(30d) | where isnotempty(RegistryKey) and isnotempty(RegistryValueName) | limit 5"
}
```