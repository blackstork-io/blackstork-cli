---
title: "`csv` data source"
plugin:
  name: blackstork/builtin
  description: "Loads CSV files with the names that match provided `glob` pattern or a single file from a provided path"
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/builtin/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v0.4.2" "csv" "data source" >}}

## Description

Loads CSV files with the names that match provided `glob` pattern or a single file from a provided path.

Either `glob` or `path` argument must be set.

When `path` argument is specified, the data source returns only the content of a file.
When `glob` argument is specified, the data source returns a list of dicts that contain the content of a file and file's metadata.

**Note**: the data source assumes that CSV file has a header: the data source turns each line into a map with the column titles as keys.

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

When `glob` is used and multiple files match the pattern, the data source will return a list of dicts, for example:

```json
[
  {
    "file_path": "path/file-a.csv",
    "file_name": "file-a.csv",
    "content": [
      {"column_A": "Foo", "column_B": true, "column_C": 42},
      {"column_A": "Bar", "column_B": false, "column_C": 4.2}
    ]
  },
  {
    "file_path": "path/file-b.csv",
    "file_name": "file-b.csv",
    "content": [
      {"column_C": "Baz", "column_D": 1},
      {"column_C": "Clu", "column_D": 2}
    ]
  },
]
```

The `csv` data source is built into the BlackStork engine. It is available out-of-the-box
and requires no installation or dependency declaration.

## Configuration

This data source accepts the following configuration arguments within a `config data csv` block:

```hcl
config data csv {
  # CSV field delimiter
  #
  # Optional string.
  # Must have a length of 1
  # Default value:
  delimiter = ","
}
```

## Usage

This data source accepts the following arguments within a `data csv` block:

```hcl
data csv {
  # A glob pattern to select CSV files to read
  #
  # Optional string.
  #
  # For example:
  # glob = "path/to/file*.csv"
  #
  # Default value:
  glob = null

  # A file path to a CSV file to read
  #
  # Optional string.
  #
  # For example:
  # path = "path/to/file.csv"
  #
  # Default value:
  path = null
}
```