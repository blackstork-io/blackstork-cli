---
title: "`stdout` publisher"
plugin:
  name: blackstork/builtin
  description: "Publishes content to stdout"
  tags: []
  version: "v0.4.2"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/internal/builtin/"
resource:
  type: publisher
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v0.4.2" "stdout" "publisher" >}}

The `stdout` publisher is built into the BlackStork engine. It is available out-of-the-box and requires no installation or dependency declaration.

## Supported Formats

This publisher supports the delivery of documents processed by the following formatters:

- `md`
- `html`

To specify the format, use the `format` argument inside the `publish` block to reference a specific `format` block or a formatter short name.


## Configuration

This publisher does not accept any configuration arguments.

## Usage

This publisher accepts the following arguments within a `publish stdout` block:

```hcl
# Note: The `publish` block also accepts the generic `format` argument to link to a formatter.

publish stdout {
}

```


