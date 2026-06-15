---
title: Data Blocks
description: Define data queries to extract structured data from external sources into the BlackStork evaluation context.
type: docs
weight: 50
---

# Data blocks

The `data` block defines a query to extract structured data from an external source. When the BlackStork engine evaluates a template, it executes the data blocks and stores the resulting JSON structures in the evaluation context. 

The block signature requires both the data source name and a block name. Together, they create a unique identifier for the block.

```hcl
# Root-level data block
data <data-source-name> "<block-name>" {
  # ...
}

document "example" {

  # In-document data block
  data <data-source-name> "<block-name>" {
    # ...
  }

}
```

You must place `data` blocks either at the root level of the configuration file or at the root level of a `document` block.

Once evaluated, the data is available in the context under the `.data.<data-source-name>.<block-name>` key. See [Evaluation Context]({{< ref "context.md" >}}) for details on accessing this data.

## Supported arguments

A `data` block accepts both generic arguments and arguments specific to the selected data source.

### Generic arguments

- `config`: (optional) A string referencing a named `config` block. If provided, the engine uses this configuration instead of the default configuration for the data source. See [Block configuration]({{< ref "configs.md#block-configuration" >}}) for details.

### Data source arguments

Arguments required to execute the query (such as endpoints, search strings, or file paths) depend strictly on the data source being used. Refer to the specific [Data Sources]({{< ref "../plugins/data-sources.md" >}}) documentation for a list of supported arguments.

## Supported blocks

- `meta`: (optional) Defines metadata for the data block. See [Metadata]({{< ref "configs.md#metadata" >}}).
- `config`: (optional) Defines an inline configuration for the data block. If provided, this block takes highest precedence, overriding both the `config` argument and the default configuration for the data source.

## References

You can reuse a standalone data block defined elsewhere in your configuration by using a `ref` block. This prevents you from writing duplicate integration definitions. See [References]({{< ref "references.md" >}}) for details on referencing blocks.

## Example

```hcl
# Default configuration for the CSV data source
config data csv {
  delimiter = ";"
}

# Root-level data block
data csv "events_a" {
  path = "/tmp/events-a.csv"
}

document "test-document" {

  # Reusing the root-level data block inside the document
  data ref {
    base = data.csv.events_a
  }

  # In-document data block with an inline configuration override
  data csv "events_b" {
    config {
      delimiter = ","
    }

    path = "/tmp/events-b.csv"
  }
}
```

## Next steps

See [Content Blocks]({{< ref "content-blocks.md" >}}) to learn how to process structured data into text, tables, and charts.

