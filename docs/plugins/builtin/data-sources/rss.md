---
title: "`rss` data source"
plugin:
  name: blackstork/builtin
  description: "Fetches RSS / Atom / JSON feed from a provided URL"
  tags: ["rss","http"]
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/builtin/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v0.4.2" "rss" "data source" >}}

## Description

Fetches RSS / Atom / JSON feed from a provided URL.

The full content of the items can be fetched and added to the feed. The data source supports basic authentication.


The `rss` data source is built into the BlackStork engine. It is available out-of-the-box
and requires no installation or dependency declaration.

## Configuration

This data source does not accept any configuration arguments.

## Usage

This data source accepts the following arguments within a `data rss` block:

```hcl
data rss {
  # Basic authentication credentials to be used in a HTTP request fetching RSS feed.
  #
  # Optional
  basic_auth {
    # Required string.
    #
    # For example:
    username = "user@example.com"

    # Note: avoid storing credentials in the templates. Use environment variables instead.
    #
    # Required string.
    #
    # For example:
    password = "passwd"
  }


  # Required string.
  #
  # For example:
  url = "https://www.elastic.co/security-labs/rss/feed.xml"

  # If the full content should be added when it's not present in the feed items.
  #
  # Optional bool.
  # Default value:
  fill_in_content = false

  # If the HTTP errors should be ignored. If set to "true", the data source will return
  # an empty dict when the endpoint returns a non-success HTTP status code.
  #
  # Optional bool.
  # Default value:
  ignore_failures = false

  # If the data source should pretend to be a browser while fetching a feed and the feed items.
  # If set to "false", the default user-agent value "blackstork-rss/0.0.1" will be used.
  #
  # Optional bool.
  # Default value:
  use_browser_user_agent = true

  # Maximum number of items to fill the content in per feed.
  #
  # Optional number.
  # Must be >= 0
  #
  # For example:
  # max_items_to_fill = 10
  #
  # Default value:
  max_items_to_fill = 10

  # Return only items published after a specified timestamp. The timestamp format is "%Y-%m-%dT%H:%M:%S%Z".
  #
  # Optional string.
  #
  # For example:
  # items_after = "2024-12-23T00:00:00Z"
  #
  # Default value:
  items_after = null

  # Return only items published before a specified timestamp. The timestamp format is "%Y-%m-%dT%H:%M:%S%Z".
  #
  # Optional string.
  #
  # For example:
  # items_before = "2024-12-23T00:00:00Z"
  #
  # Default value:
  items_before = null
}
```