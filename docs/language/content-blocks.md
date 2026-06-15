---
title: Content Blocks
description: Use content blocks to process structured data into text, tables, and AI-generated narratives within your BlackStork templates.
type: docs
weight: 60
---

# Content blocks

The `content` block defines the specific segments of your report, such as text paragraphs, tables, charts, or lists. 

The engine renders `content` blocks in the exact order they are declared within the template, establishing the flow of your final document.

The block signature requires the name of the content provider that will execute the block.


```hcl
# Root-level named content block
content <content-provider> "<block-name>" {
  # ...
}

document "foobar" {

  # In-document named content block
  content <content-provider> "<block-name>" {
    # ...
  }

  # In-document anonymous content block
  content <content-provider> {
    # ...
  }

}
```

If you define a `content` block at the root level of the file (outside of a `document` or `section` block), you must provide a block name. The combination of block type (`content`), provider name, and block name creates a unique identifier allowing you to reference the block elsewhere.

If you define the block directly within a `document` or `section`, the block name is optional.

Refer to [Content Providers]({{< ref "content-providers.md" >}}) for a complete list of supported rendering providers.

## Supported arguments

A `content` block accepts both generic arguments and arguments specific to the selected content provider.

### Generic arguments

- `config`: (optional) A string referencing a named `config` block. If provided, the engine uses this configuration instead of the default configuration for the provider. See [Block configuration]({{< ref "configs.md#block-configuration" >}}) for details.
- `local_var`: (optional) A shortcut to define a single local variable named `local` within the block's scope. See [Local variable]({{< ref "context.md#local-variable" >}}).

### Content provider arguments

Arguments determining what the block renders (such as template strings, prompts, or array mappings) depend strictly on the content provider. Refer to the specific [Content Providers]({{< ref "content-providers.md" >}}) documentation for the supported arguments.

## Supported blocks

- `meta`: (optional) Defines metadata for the content block. See [Metadata]({{< ref "configs.md#metadata" >}}).
- `config`: (optional) Defines an inline configuration for the block. If provided, this block takes highest precedence, overriding both the `config` argument and the default configuration for the content provider.
- `vars`: (optional) Defines variables within the scope of the content block. The engine evaluates these variables before executing the provider, allowing you to manipulate context data before injecting it into the content. See [Variables]({{< ref "context.md#variables" >}}).

## References

You can reuse a content block defined elsewhere in your configuration by using a `ref` block. See [References]({{< ref "references.md" >}}) for details.

## Example

```hcl
# Named configuration for a specific OpenAI account
config content openai_text "test_account" {
  # Reading a key from an environment variable
  api_key = env.OPENAI_API_KEY
}

document "test-doc" {

  vars {
    items = ["aaa", "bbb", "ccc"]
  }

  content text {
    # Extract the length of the array into a local variable using JQ
    local_var = query_jq(".vars.items | length")

    # Inject variables directly into the Markdown string using Go templates
    value = "There are {{ .vars.local }} items: {{ .vars.items | toPrettyJson }}"
  }

  content openai_text {
    # Reference the explicit configuration block defined above
    config = config.content.openai_text.test_account

    # Pass the context variable into the LLM prompt
    prompt = <<-EOT
       Write a short story, just a paragraph, about space exploration
       using the values from the provided items list as character names:

       {{ .vars.items | toPrettyJson }}
    EOT
  }
}
```

The engine processes the static text block and queries the OpenAI API for the second block, producing the following output:

```text
There are 3 items: [
  "aaa",
  "bbb",
  "ccc"
]

In the vast expanse of outer space, aaa, bbb, and ccc embarked on a daring mission of exploration. Their spaceship soared through the galaxies, encountering unknown planets and celestial bodies. With aaa's courage leading the way, bbb's wisdom guiding their decisions, and ccc's ingenuity solving the complex challenges they faced, the trio delved deeper into the mysteries of the cosmos.
```

## Next steps

See [Section Blocks]({{< ref "section-blocks.md" >}}) to learn how to organize content blocks into logical chapters and iterate over data arrays.
