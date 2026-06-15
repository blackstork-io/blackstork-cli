---
title: "`txt` data source"
plugin:
  name: blackstork/builtin
  description: "Loads TXT files with the names that match provided `glob` pattern or a single file from a provided path"
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/builtin/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v0.4.2" "txt" "data source" >}}

## Description

Loads TXT files with the names that match provided `glob` pattern or a single file from a provided path.

Either `glob` or `path` argument must be set.

When `path` argument is specified, the data source returns only the content of a file.
When `glob` argument is specified, the data source returns a list of dicts that contain the content of a file and file's metadata. For example:
```json
[
  {
    "file_path": "path/file-a.txt",
    "file_name": "file-a.txt",
    "content": "foobar"
  },
  {
    "file_path": "path/file-b.txt",
    "file_name": "file-b.txt",
    "content": "x\\ny\\nz"
  }
]
```

The `txt` data source is built into the BlackStork engine. It is available out-of-the-box
and requires no installation or dependency declaration.

## Configuration

This data source does not accept any configuration arguments.

## Usage

This data source accepts the following arguments within a `data txt` block:

```hcl
data txt {
  # A glob pattern to select TXT files to read
  #
  # Optional string.
  #
  # For example:
  # glob = "path/to/file*.txt"
  #
  # Default value:
  glob = null

  # A file path to a TXT file to read
  #
  # Optional string.
  #
  # For example:
  # path = "path/to/file.txt"
  #
  # Default value:
  path = null
}
```