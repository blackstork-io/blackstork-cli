---
title: "`http` data source"
plugin:
  name: blackstork/builtin
  description: "Loads data from a URL"
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

Loads data from a URL.

Accepts only responses with UTF-8 charset with MIME types `text/csv`, `application/json` or `application/yaml` are parsed.
A correct supported MIME type must be either set explicitly in `response_mime_type` or provided in the HTTP response header.
If received MIME type is not supported, the response content will be returned as plain text.

Response content is parsed and returned as a JSON object (similar to `csv`, `json` and `yaml`
data sources).


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

  # Value to override response MIME type with. Supported values: `application/json`, `text/csv` and `application/yaml`. If not provided, an original MIME type from the response will be used.
  #
  # Optional string.
  # Default value:
  response_mime_type = null
}
```