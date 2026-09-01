---
title: "`snyk_issues` data source"
plugin:
  name: blackstork/snyk
  description: "The `snyk_issues` data source fetches issues from Snyk"
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/snyk/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/snyk" "snyk" "v1.0.0-rc1" "snyk_issues" "data source" >}}

## Description
The `snyk_issues` data source fetches issues from Snyk.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `snyk_issues` data source locally via `blackstork-cli`, you must declare the `blackstork/snyk` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/snyk" = ">= v1.0.0-rc1"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data snyk_issues` block:

```hcl
config data snyk_issues {
  # The Snyk API key
  #
  # Required string.
  # Must be non-empty
  #
  # For example:
  api_key = "some string"
}
```

## Usage

This data source accepts the following arguments within a `data snyk_issues` block:

```hcl
data snyk_issues {
  # The group ID
  #
  # Optional string.
  # Default value:
  group_id = null

  # The organization ID
  #
  # Optional string.
  # Default value:
  org_id = null

  # The scan item ID
  #
  # Optional string.
  # Default value:
  scan_item_id = null

  # The scan item type
  #
  # Optional string.
  # Default value:
  scan_item_type = null

  # The issue type
  #
  # Optional string.
  # Default value:
  type = null

  # The updated before date
  #
  # Optional string.
  # Default value:
  updated_before = null

  # The updated after date
  #
  # Optional string.
  # Default value:
  updated_after = null

  # The created before date
  #
  # Optional string.
  # Default value:
  created_before = null

  # The created after date
  #
  # Optional string.
  # Default value:
  created_after = null

  # The effective severity level
  #
  # Optional list of string.
  # Default value:
  effective_severity_level = null

  # The status
  #
  # Optional list of string.
  # Default value:
  status = null

  # The ignored flag
  #
  # Optional bool.
  # Default value:
  ignored = null

  # The limit of issues to fetch
  #
  # Optional number.
  # Must be >= 0
  # Default value:
  limit = 0
}
```