---
title: Publish Blocks
description: Use publish blocks to format, route, and deliver rendered BlackStork documents to local filesystems or external integrations.
type: docs
weight: 75
---

# Publish blocks

The `publish` block handles the final step of the BlackStork execution pipeline. After the engine extracts structured data, evaluates content logic, and applies visual styling via `format` blocks, `publish` blocks dictate how and where that final document is delivered.

A publisher can write output to a local filesystem, push it to a Git repository, upload it to cloud storage, or send it to external systems like Confluence or ticketing platforms. 

The block signature requires the name of the publisher component.

```hcl
document "example" {

  # In-document named publish block
  publish <publisher-name> "<block-name>" {
    # ...
  }

  # In-document anonymous publish block
  publish <publisher-name> {
    # ...
  }

}
```

If you define a `publish` block at the root level of the file (outside of a `document` block), you must provide a block name. The combination of block type (`publish`), publisher name, and block name creates a unique identifier. If defined within a `document`, the block name is optional.

A single `document` block can contain multiple `publish` blocks. During execution, the engine picks and execute all `publish` blocks or a specific `publish` block requested by user.

Within a `document`, `publish` blocks can be defined only on the root level.

Refer to [Publishers]({{< ref "publishers.md" >}}) for a complete list of supported delivery integrations.

## Execution

By default, executing `blackstork-cli render` outputs the evaluated document directly to standard output (`stdout`) as Markdown. 

To execute the `publish` blocks defined in your configuration, you must explicitly pass the `--publish` flag to the CLI.

```bash
# Output the document to stdout as Markdown (default)
$ blackstork-cli render document.example

# Execute all publish blocks defined in the document template
$ blackstork-cli render document.example --publish
```

## Linking publishers to formatters

Publishers do not format documents; they only deliver them. A `publish` block must receive a rendered payload from a `format` block. Furthermore, individual publishers only support specific file formats (e.g., a GitHub Gist publisher may only accept `md`).

You control which formatted output is sent to the publisher using the `format` argument within the `publish` block. The engine resolves this argument using the following rules:

1. **Full Block Name (Recommended):** `format = "format.html.styled_report"`
   The publisher explicitly consumes the output of the named `format` block defined in the document. This is the safest and most predictable method.
2. **Short Name:** `format = "html"`
   The engine searches the document for the first `format` block of the specified type (e.g., `format html`). If a matching block is found, its output is used. If no matching block exists in the document, the engine executes the default `html` formatter.
3. **Omitted:** 
   If the `format` argument is omitted entirely, the engine picks the first `format` block in the document that the publisher supports. If no format blocks are defined, it falls back to the default configuration. **Note:** If a publisher supports multiple formats, this behavior is highly unpredictable. Explicitly defining the `format` argument is strongly recommended.

## Supported arguments

A `publish` block accepts both generic arguments and arguments specific to the selected publisher.

### Generic arguments

- `format`: (optional) A string indicating the formatter output to consume. Can be a short name (`md`, `html`, `pdf`) or a full block reference (`format.pdf.executive_summary`) resolvable inside the document. 
- `config`: (optional) A string referencing a named `config` block. If provided, the engine uses this configuration instead of the default configuration for the publisher. See [Block configuration]({{< ref "configs.md#block-configuration" >}}) for details.

### Publisher arguments

Arguments determining delivery behavior (such as file paths, API endpoints, or repository details) depend strictly on the publisher. Refer to the specific [Publishers]({{< ref "publishers.md" >}}) documentation for supported arguments.

## Supported blocks

- `meta`: (optional) Defines metadata for the publish block. See [Metadata]({{< ref "configs.md#metadata" >}}).
- `config`: (optional) Defines an inline configuration for the block. If provided, this block takes highest precedence, overriding both the `config` argument and the default configuration for the publisher.

## Example

The following configuration explicitly defines two formatters and two delivery destinations using the `local_file` publisher. 

Note that the `local_file` publisher treats its `path` argument as a Go template string, allowing you to dynamically generate filenames.

```hcl
document "threat_brief" {
  title = "Daily Threat Brief"

  content text {
    value = "Static text in the document body"
  }

  # Define the formats:
  format pdf "executive_pdf" {
    # PDF styling configurations...
  }

  format html "web_view" {
    # HTML styling configurations...
  }

  # Define the publishers and link them to the formats:
  publish local_file "pdf_out" {
    format = "format.pdf.executive_pdf"
    path   = "docs/threat_brief_{{ now | date \"2006_01_02\" }}.pdf"
  }

  publish local_file "html_out" {
    format = "format.html.web_view"
    path   = "html/threat_brief_latest.html"
  }
}
```

```bash
$ blackstork-cli render document.threat_brief --publish
Jun  2 12:48:08.899 INF Writing to a file path=docs/threat_brief_2024_06_02.pdf
Jun  2 12:48:09.182 INF Writing to a file path=html/threat_brief_latest.html
```

## Next steps

See [References]({{< ref "references.md" >}}) to learn how to reuse structural blocks, such as standardized headers or publishing destinations, across multiple templates.
