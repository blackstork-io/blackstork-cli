---
title: "`http` data source"
plugin:
  name: blackstork/builtin
  description: "Fetches HTTP response from URL, parses its body and loads it"
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/builtin/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v0.4.2" "http" "data source" >}}

## Description

Fetches HTTP response from URL, parses its body and loads it.

If the format of the response is not supported, data source will return response body as plain text.
If format is supported, response body will be parsed and returned as a JSON object (fimilar to `file` data source).


The `http` data source is built into the BlackStork engine. It is available out-of-the-box
and requires no installation or dependency declaration.

## Configuration

This data source does not accept any configuration arguments.

## Usage

This data source accepts the following arguments within a `data http` block:

```hcl
data http {
  # Basic authentication credentials to be used for HTTP request.
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


  # URL to fetch data from. Supported schemas are `http` and `https`
  #
  # Required string.
  # Must be non-empty
  #
  # For example:
  url = "https://example.localhost/file.json"

  # HTTP method for the request. Allowed methods are `GET`, `POST` and `HEAD`
  #
  # Optional string.
  # Must be one of: "GET", "POST", "HEAD"
  # Default value:
  method = "GET"

  # If set to `true`, disabled verification of the server's certificate.
  #
  # Optional bool.
  # Default value:
  insecure = false

  # The duration of a timeout for a request. Accepts numbers, with optional fractions and a unit suffix. For example, valid values would be: 1.5s, 30s, 2m, 2m30s, or 1h
  #
  # Optional string.
  # Default value:
  timeout = "30s"

  # The headers to be set in a request
  #
  # Optional map of string.
  # Default value:
  headers = null

  # Request body
  #
  # Optional string.
  # Default value:
  body = null

  # If provided, overrides response MIME type. If not provided, the format is deduced from response MIME type.
  #
  # Optional string.
  # Must be one of: "json", "yaml", "csv", "text", null
  # Default value:
  format = null
}
```