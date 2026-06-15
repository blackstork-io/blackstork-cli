---
title: "`hackerone_reports` data source"
plugin:
  name: blackstork/hackerone
  description: ""
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/hackerone/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/hackerone" "hackerone" "v0.4.2" "hackerone_reports" "data source" >}}

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `hackerone_reports` data source locally via `blackstork-cli`, you must declare the `blackstork/hackerone` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/hackerone" = ">= v0.4.2"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data hackerone_reports` block:

```hcl
config data hackerone_reports {
  # Required string.
  #
  # For example:
  api_username = "some string"

  # Required string.
  #
  # For example:
  api_token = "some string"
}
```

## Usage

This data source accepts the following arguments within a `data hackerone_reports` block:

```hcl
data hackerone_reports {
  # Optional number.
  # Default value:
  size = null

  # Optional number.
  # Default value:
  page_number = null

  # Optional string.
  # Default value:
  sort = null

  # Optional list of string.
  # Default value:
  program = null

  # Optional list of number.
  # Default value:
  inbox_ids = null

  # Optional list of string.
  # Default value:
  reporter = null

  # Optional list of string.
  # Default value:
  assignee = null

  # Optional list of string.
  # Default value:
  state = null

  # Optional list of number.
  # Default value:
  id = null

  # Optional list of number.
  # Default value:
  weakness_id = null

  # Optional list of string.
  # Default value:
  severity = null

  # Optional bool.
  # Default value:
  hacker_published = null

  # Optional string.
  # Default value:
  created_at__gt = null

  # Optional string.
  # Default value:
  created_at__lt = null

  # Optional string.
  # Default value:
  submitted_at__gt = null

  # Optional string.
  # Default value:
  submitted_at__lt = null

  # Optional string.
  # Default value:
  triaged_at__gt = null

  # Optional string.
  # Default value:
  triaged_at__lt = null

  # Optional bool.
  # Default value:
  triaged_at__null = null

  # Optional string.
  # Default value:
  closed_at__gt = null

  # Optional string.
  # Default value:
  closed_at__lt = null

  # Optional bool.
  # Default value:
  closed_at__null = null

  # Optional string.
  # Default value:
  disclosed_at__gt = null

  # Optional string.
  # Default value:
  disclosed_at__lt = null

  # Optional bool.
  # Default value:
  disclosed_at__null = null

  # Optional bool.
  # Default value:
  reporter_agreed_on_going_public = null

  # Optional string.
  # Default value:
  bounty_awarded_at__gt = null

  # Optional string.
  # Default value:
  bounty_awarded_at__lt = null

  # Optional bool.
  # Default value:
  bounty_awarded_at__null = null

  # Optional string.
  # Default value:
  swag_awarded_at__gt = null

  # Optional string.
  # Default value:
  swag_awarded_at__lt = null

  # Optional bool.
  # Default value:
  swag_awarded_at__null = null

  # Optional string.
  # Default value:
  last_report_activity_at__gt = null

  # Optional string.
  # Default value:
  last_report_activity_at__lt = null

  # Optional string.
  # Default value:
  first_program_activity_at__gt = null

  # Optional string.
  # Default value:
  first_program_activity_at__lt = null

  # Optional bool.
  # Default value:
  first_program_activity_at__null = null

  # Optional string.
  # Default value:
  last_program_activity_at__gt = null

  # Optional string.
  # Default value:
  last_program_activity_at__lt = null

  # Optional bool.
  # Default value:
  last_program_activity_at__null = null

  # Optional string.
  # Default value:
  last_activity_at__gt = null

  # Optional string.
  # Default value:
  last_activity_at__lt = null

  # Optional string.
  # Default value:
  last_public_activity_at__gt = null

  # Optional string.
  # Default value:
  last_public_activity_at__lt = null

  # Optional string.
  # Default value:
  keyword = null

  # Optional map of string.
  # Default value:
  custom_fields = null
}
```