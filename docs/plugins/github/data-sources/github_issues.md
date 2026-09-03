---
title: "`github_issues` data source"
plugin:
  name: blackstork/github
  description: "Retrieves issues from a GitHub repository. Supports filtering by milestone, assignee, creator, mentioned user, labels, state, and an updated-since timestamp"
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/github/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/github" "github" "v1.0.0" "github_issues" "data source" >}}

## Description
Retrieves issues from a GitHub repository. Supports filtering by milestone, assignee, creator, mentioned user, labels, state, and an updated-since timestamp.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `github_issues` data source locally via `blackstork-cli`, you must declare the `blackstork/github` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/github" = ">= v1.0.0"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data github_issues` block:

```hcl
config data github_issues {
  # The GitHub token to use for authentication
  #
  # Required string.
  # Must be non-empty
  #
  # For example:
  github_token = "some string"
}
```

## Usage

This data source accepts the following arguments within a `data github_issues` block:

```hcl
data github_issues {
  # The repository to list issues from, in the format of owner/name
  #
  # Required string.
  # Must be non-empty
  #
  # For example:
  repository = "blackstork-io/blackstork-cli"

  # Filter issues by milestone. Possible values are:
  # * a milestone number
  # * "none" for issues with no milestone
  # * "*" for issues with any milestone
  # * "" (empty string) performs no filtering
  #
  # Optional string.
  # Default value:
  milestone = ""

  # Filter issues based on their state
  #
  # Optional string.
  # Must be one of: "open", "closed", "all"
  # Must be non-empty
  # Default value:
  state = "open"

  # Filter issues based on their assignee. Possible values are:
  # * a user name
  # * "none" for issues that are not assigned
  # * "*" for issues with any assigned user
  # * "" (empty string) performs no filtering.
  #
  # Optional string.
  # Default value:
  assignee = ""

  # Filter issues based on their creator. Possible values are:
  # * a user name
  # * "" (empty string) performs no filtering.
  #
  # Optional string.
  # Default value:
  creator = ""

  # Filter issues to once where this username is mentioned. Possible values are:
  # * a user name
  # * "" (empty string) performs no filtering.
  #
  # Optional string.
  # Default value:
  mentioned = ""

  # Filter issues based on their labels.
  #
  # Optional list of string.
  # Default value:
  labels = null

  # Specifies how to sort issues.
  #
  # Optional string.
  # Must be one of: "created", "updated", "comments"
  # Must be non-empty
  # Default value:
  sort = "created"

  # Specifies the direction in which to sort issues.
  #
  # Optional string.
  # Must be one of: "asc", "desc"
  # Must be non-empty
  # Default value:
  direction = "desc"

  # Only show results that were last updated after the given time.
  # This is a timestamp in ISO 8601 format: YYYY-MM-DDTHH:MM:SSZ.
  #
  # Optional string.
  # Must be non-empty
  # Default value:
  since = null

  # Limit the number of issues to return. -1 means no limit.
  #
  # Optional integer.
  # Must be >= -1
  # Default value:
  limit = -1
}
```