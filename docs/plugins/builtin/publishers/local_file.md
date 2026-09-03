---
title: "`local_file` publisher"
plugin:
  name: blackstork/builtin
  description: "Writes the rendered document to a local file."
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/builtin/"
resource:
  type: publisher
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/builtin" "builtin" "v1.0.0-rc1" "local_file" "publisher" >}}

## Description
Writes the rendered document to a local file.

The `local_file` publisher is built into the BlackStork engine. It is available out-of-the-box and requires no installation or dependency declaration.


## Configuration

This publisher does not accept any configuration arguments.

## Usage

This publisher accepts the following arguments within a `publish local_file` block:

```hcl
# The `publish` block also accepts the generic `format_ref` argument to link to a formatter.

publish local_file {
  # Path to the file
  #
  # Required string.
  #
  # For example:
  path = "dist/output.md"
}

```

