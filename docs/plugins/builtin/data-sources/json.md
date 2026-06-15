---
title: "`json` data source"
plugin:
  name: blackstork/builtin
  description: "Loads JSON files with the names that match provided `glob` pattern or a single file from provided `path`value"
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/builtin/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v0.4.2" "json" "data source" >}}

## Description

Loads JSON files with the names that match provided `glob` pattern or a single file from provided `path`value.

Either `glob` or `path` argument must be set.

When `path` argument is specified, the data source returns only the content of a file.
When `glob` argument is specified, the data source returns a list of dicts that contain the content of a file and file's metadata. For example:

```json
[
  {
    "file_path": "path/file-a.json",
    "file_name": "file-a.json",
    "content": {
      "foo": "bar"
  }
  },
  {
    "file_path": "path/file-b.json",
    "file_name": "file-b.json",
    "content": [
      {"x": "y"}
    ]
  }
]
```

The `json` data source is built into the BlackStork engine. It is available out-of-the-box
and requires no installation or dependency declaration.

## Configuration

This data source does not accept any configuration arguments.

## Usage

This data source accepts the following arguments within a `data json` block:

```hcl
data json {
  # A glob pattern to select JSON files to read
  #
  # Optional string.
  #
  # For example:
  # glob = "path/to/file*.json"
  #
  # Default value:
  glob = null

  # A file path to a JSON file to read
  #
  # Optional string.
  #
  # For example:
  # path = "path/to/file.json"
  #
  # Default value:
  path = null
}
```