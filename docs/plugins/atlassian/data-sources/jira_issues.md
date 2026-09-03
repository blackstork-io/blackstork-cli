---
title: "`jira_issues` data source"
plugin:
  name: blackstork/atlassian
  description: "Retrieves Jira Cloud issues that match the supplied JQL and field-selection options. Returns a list of issue objects"
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/atlassian/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/atlassian" "atlassian" "v1.0.0" "jira_issues" "data source" >}}

## Description
Retrieves Jira Cloud issues that match the supplied JQL and field-selection options. Returns a list of issue objects.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `jira_issues` data source locally via `blackstork-cli`, you must declare the `blackstork/atlassian` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/atlassian" = ">= v1.0.0"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data jira_issues` block:

```hcl
config data jira_issues {
  # Account Domain.
  #
  # Required string.
  # Must be non-empty
  #
  # For example:
  domain = "some string"

  # Account Email.
  #
  # Required string.
  # Must be non-empty
  #
  # For example:
  account_email = "some string"

  # API Token.
  #
  # Required string.
  # Must be non-empty
  #
  # For example:
  api_token = "some string"
}
```

## Usage

This data source accepts the following arguments within a `data jira_issues` block:

```hcl
data jira_issues {
  # Use expand to include additional information about issues in the response.
  #
  # Optional string.
  # Must be one of: "renderedFields", "names", "schema", "changelog"
  #
  # For example:
  # expand = "names"
  #
  # Default value:
  expand = null

  # A list of fields to return for each issue.
  #
  # Optional list of string.
  #
  # For example:
  # fields = ["*all"]
  #
  # Default value:
  fields = null

  # A JQL expression. For performance reasons, this field requires a bounded query. A bounded query is a query with a search restriction.
  #
  # Optional string.
  #
  # For example:
  # jql = "order by key desc"
  #
  # Default value:
  jql = null

  # A list of up to 5 issue properties to include in the results.
  #
  # Optional list of string.
  # Must contain no more than 5 elements.
  # Default value:
  properties = []

  # Size limit to retrieve.
  #
  # Optional number.
  # Must be >= 0
  # Default value:
  size = 0
}
```