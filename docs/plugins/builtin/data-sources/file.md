---
title: "`file` data source"
plugin:
  name: blackstork/builtin
  description: "Reads local JSON, YAML, CSV, or text files and returns their contents as template data"
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/builtin/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v1.0.0" "file" "data source" >}}

## Description

Reads local JSON, YAML, CSV, or text files and returns their contents as template data.

Set either `path` to read one file or `glob` to read every matching file.
With `path`, the data source returns the decoded file contents directly. With `glob`,
it returns a list containing each file's path, name, and decoded contents:

```json
[
  {
    "file_path": "path/file-a.json",
    "file_name": "file-a.json",
    "content": {"foo": "bar"}
  }
]
```

All files matched by a glob are decoded using the selected `format`. CSV input must
include a header row; each subsequent row is returned as an object keyed by the column names.


The `file` data source is built into the BlackStork engine. It is available out-of-the-box
and requires no installation or dependency declaration.

## Configuration

This data source does not accept any configuration arguments.

## Usage

This data source accepts the following arguments within a `data file` block:

```hcl
data file {
  # A glob pattern to select files to read
  #
  # Optional string.
  #
  # For example:
  # glob = "path/to/file*.json"
  #
  # Default value:
  glob = null

  # A file path to a file to read
  #
  # Optional string.
  #
  # For example:
  # path = "path/to/file.yaml"
  #
  # Default value:
  path = null

  # File format
  #
  # Required string.
  # Must be one of: "json", "yaml", "csv", "text"
  # Must be non-empty
  #
  # For example:
  format = "json"

  # CSV field delimiter
  #
  # Optional string.
  # Must have a length of 1
  # Default value:
  csv_delimiter = ","
}
```