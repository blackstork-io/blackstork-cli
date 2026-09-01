---
title: "`file` data source"
plugin:
  name: blackstork/builtin
  description: "Loads files with the names that match provided `glob` pattern or a single file from provided `path`value"
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/builtin/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v1.0.0-rc1" "file" "data source" >}}

## Description

Loads files with the names that match provided `glob` pattern or a single file from provided `path`value.

Either `glob` or `path` must be set.

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

If multiple files are matched, all files must have the same file format.

For CSV files, the data source assumes that CSV file has a header: each line of the file is turned into a map with the column titles as keys.

For example, CSV file with the following data:

| column_A | column_B | column_C |
| -------- | -------- | -------- |
| Foo      | true     | 42       |
| Bar      | false    | 4.2      |

will be represented as the following data structure:
```json
[
  {"column_A": "Foo", "column_B": true, "column_C": 42},
  {"column_A": "Bar", "column_B": false, "column_C": 4.2}
]
```


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