---
title: "`table` content provider"
plugin:
  name: blackstork/builtin
  description: "Renders data as a table using a Go template for each column header and cell"
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/builtin/"
resource:
  type: content-provider
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v1.0.0" "table" "content provider" >}}

## Description

Renders data as a table using a Go template for each column header and cell.

Each cell template has access to the document data context and these variables:

* `.rows` – the value of `rows` argument
* `.row.value` – the current row from `.rows` list
* `.row.index` – the current row index
* `.col.index` – the current column index

Header templates have access to the document data context, `.rows`, and `.col.index`,
but not `.row.value` or `.row.index`.

The `table` content provider is built into the BlackStork engine. It is available out-of-the-box and requires no installation or dependency declaration.

## Configuration

This content provider does not accept any configuration arguments.

## Usage

This content provider accepts the following arguments within a `content table` block:

```hcl
content table {
  # A list of objects representing rows in the table. Can be a static list or `query_jq()` func call.
  #
  # Optional list of jq queriable.
  # Default value:
  rows = null

  # List of a header and a cell template pairs for each column
  #
  # Required list of object.
  # Must be non-empty
  #
  # For example:
  columns = [{
    header = "1st column header template"
    value  = "1st column value template"
    }, {
    header = "2nd column header template"
    value  = "2nd column value template"
  }]
}
```

