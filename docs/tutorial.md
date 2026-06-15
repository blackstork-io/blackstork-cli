---
title: Tutorial
description: A step-by-step guide to writing BlackStork templates. Learn how to query data, use variables, and render documents locally with blackstork-cli.
type: docs
weight: 10
---

# Tutorial

This tutorial covers the core mechanics of the BlackStork configuration language. We will create a basic template, define variables, install plugins, and generate text using an external content provider (OpenAI). 

While these templates can be evaluated collaboratively within the BlackStork SaaS platform, this tutorial uses `blackstork-cli` to evaluate the templates and render the documents locally.

## Prerequisites

- The `blackstork-cli` binary [installed]({{< ref "install.md" >}}) and available in your `PATH`.
- (Optional) An OpenAI API token.

## Basic Template

Create a basic template to verify the execution environment. Create a new file named `hello.blackstork.hcl` and define a `document` block:

```hcl
document "greeting" {

  content text {
    value = "Hello, BlackStork!"
  }

}
```

The `document.greeting` block defines a template containing a single anonymous content block.

To evaluate the template and render the document, execute the CLI in the directory containing your file:

```shell
blackstork-cli render document.greeting
```

The engine evaluates the block and outputs the string to standard output:

```shell
$ blackstork-cli render document.greeting
Hello, BlackStork!
```

## Document title

The `document` block accepts a `title` argument to set the document's top-level header. Update the `document.greeting` block:

```hcl
document "greeting" {

  title = "The Greeting"

  content text {
    value = "Hello, BlackStork!"
  }

}
```

{{< hint note >}}

The `title` argument in a `document` block is syntactic sugar. During evaluation, the engine translates it into a `content.title` block:

```hcl
content title {
  value = "The Greeting"
}
```

See the [`content.title`]({{< ref "plugins/builtin/content-providers/title" >}}) provider documentation for details.
{{< /hint >}}

The rendered Markdown output now includes the header:


```markdown
$ blackstork-cli render document.greeting
# The Greeting

Hello, BlackStork!
```

## Variables

Instead of configuring external data sources for this tutorial, use a `vars` block to define static variables.

Update `hello.blackstork.hcl` file to include a `vars` block and a second `content.text` block:

```hcl
document "greeting" {

  vars {
    solar_system = {
      planets = [
        "Mercury", "Venus", "Earth", "Mars", "Jupiter", "Saturn", "Uranus", "Neptune"
      ]
      moons_count = 146
    }
  }

  title = "The Greeting"

  content text {
    value = "Hello, BlackStork!"
  }

  content text {
    local_var = query_jq(".vars.solar_system.planets | length")

    value = <<-EOT
      There are {{ .vars.local }} planets and {{ .vars.solar_system.moons_count }} moons in our solar system.
    EOT
  }

}
```

This configuration defines inline data inside the `vars` block and creates a local variable (`local_var`,  see [Local Variable]({{< ref "context.md#local-variable" >}})) that contains the array length calculated with `jq` query using the `query_jq()` function (see [Querying the context]({{< ref "context.md#querying-the-context" >}})).

The `value` argument in the `content.text` block accepts Go templates. This allows you to inject evaluated context data directly into your text.

```shell
$ blackstork-cli render document.greeting
# The Greeting

Hello, BlackStork!

There are 8 planets and 146 moons in our solar system.
```

## Content providers

The BlackStork engine can execute local logic or integrate with external APIs to generate content. In scenarios where static templates are insufficient, you can pass structured data to an LLM to dynamically draft text.

The following steps use the [`openai_text`]({{< ref "plugins/openai/content-providers/openai_text" >}}) content provider to generate text via the OpenAI API. There is also a built-in generic [`llm_text`]({{< ref "plugins/builtin/content-providers/llm_text" >}}) content provider that supports LLM from a selection of vendors.

### Installation

Before using [`openai_text`]({{< ref "plugins/openai/content-providers/openai_text" >}}) content
provider, you must declare [`blackstork/openai`]({{< ref "plugins/openai" >}}) plugin as a
dependency and install it locally.

Add a global `blackstork` configuration block (see [Global configuration]({{< ref "language/configs.md#global-configuration" >}})) to `hello.blackstork.hcl`:

```hcl
blackstork {
  plugin_versions = {
    "blackstork/openai" = ">= 0.4.0"
  }
}

document "greeting" {

  vars {
    solar_system = {
      planets = [
        "Mercury", "Venus", "Earth", "Mars", "Jupiter", "Saturn", "Uranus", "Neptune"
      ]
      moons_count = 146
    }
  }

  title = "The Greeting"

  content text {
    value = "Hello, BlackStork!"
  }

  content text {
    local_var = query_jq(".vars.solar_system.planets | length")

    value = <<-EOT
      There are {{ .vars.local }} planets and {{ .vars.solar_system.moons_count }} moons in our solar system.
    EOT
  }

}
```

Run the install command to fetch the required plugin from the registry:

```shell
$ blackstork-cli install
Mar 11 19:20:10.769 INF Searching plugin name=blackstork/openai constraints=">=v0.4.0"
Mar 11 19:20:10.787 INF Installing plugin name=blackstork/openai version=0.4.0
$
```

The CLI downloads the plugin and places it in the local `./.blackstork/` directory.

### Configuration

The OpenAI API requires authentication. Pass your credentials using the `env` object (see [Environment variables]({{< ref "configs.md#environment-variables" >}})) rather than hardcoding them in your configuration files.

Add the content provider `config` block to the root level of `hello.blackstork.hcl`:

```hcl
config content openai_text {
  api_key = env.OPENAI_API_KEY
}
```

Now the OpenAI API key specified in `OPENAI_API_KEY` environment variable is passed on as an
argument to `api_key` attribute of the `config` block.

### Usage

Define a `content` block that calls the `openai_text` content provider:


```hcl
# ...

document "greeting" {

  # ...

  content openai_text {
    local_var = query_jq("{planet: .vars.solar_system.planets[-1]}")

    prompt = <<-EOT
      Share a fact about the planet specified in the provided data:
      {{ .vars.local | toRawJson }}
    EOT
  }

}
```

This block also uses a `jq` query to extract the last item from the `.vars.solar_system.planets` array (Neptune) and passes it to the LLM prompt as a JSON object.


{{< hint note >}}
You can define a system prompt for the OpenAI API in the provider configuration. See the `openai_text` content provider [documentation]({{< ref "plugins/openai/content-providers/openai_text" >}}) for all available options.
{{< /hint >}}

The complete `hello.blackstork.hcl` file should look like this:

```hcl
blackstork {
  plugin_versions = {
    "blackstork/openai" = ">= 0.4"
  }
}

config content openai_text {
  api_key = env.OPENAI_API_KEY
}

document "greeting" {

  vars {
    solar_system = {
      planets = [
        "Mercury", "Venus", "Earth", "Mars", "Jupiter", "Saturn", "Uranus", "Neptune"
      ]
      moons_count = 146
    }
  }

  title = "The Greeting"

  content text {
    value = "Hello, BlackStork!"
  }

  content text {
    local_var = query_jq(".vars.solar_system.planets | length")

    value = <<-EOT
      There are {{ .vars.local }} planets and {{ .vars.solar_system.moons_count }} moons in our solar system.
    EOT
  }

  content openai_text {
    local_var = query_jq("{planet: .vars.solar_system.planets[-1]}")

    prompt = <<-EOT
      Share a fact about the planet specified in the provided data:
      {{ .vars.local | toRawJson }}
    EOT
  }

}
```

Pass the `OPENAI_API_KEY` environment variable to the CLI to render the document:


```shell
$ OPENAI_API_KEY="<key>" blackstork-cli render document.greeting
...
```

{{< hint warning >}}
Remember to replace `<key>` in the CLI command with your OpenAI API key value.
{{< /hint >}}

The output will include the static strings and the dynamically generated text:

```bash
$ OPENAI_API_KEY="<key-value>" ./blackstork-cli render document.greeting
Jun 23 17:14:23.910 INF Parsing BlackStork files command=render
Jun 23 17:14:23.912 INF Loading plugin resolver command=render includeRemote=false
Jun 23 17:14:23.912 INF Loading plugin runner command=render
Jun 23 17:14:23.939 INF Rendering content command=render target=greeting
Jun 23 17:14:23.939 INF Loading document command=render target=greeting
# The Greeting

Hello, BlackStork!

There are 8 planets and 146 moons in our solar system.

Neptune is the eighth and farthest known planet from the Sun in the Solar System. It is classified as an ice giant and is the fourth-largest planet by diameter.
```

## Publishing

By default, the engine outputs the rendered Markdown to standard output. To format the output as HTML or PDF, use a `format` block, and to write it to a file or BlackStork SaaS, use `publish` block.

Add a `publish` block and a `format` block to the document template:

```hcl
document "greeting" {

  # ...

  # HTML format with default settings
  format html {
  }

  # Publishing to a local HTML file
  publish local_file {
    path = "./greeting-{{ now | date \"2006_01_02\" }}.{{.format}}"
  }

}
```

The path argument supports Go templates, allowing you to define dynamic filenames based on timestamps and format extensions.

Execute the publish block by appending the `--publish` flag to the render command:

```bash
$ blackstork-cli render document.greeting --publish
Jun 23 17:28:03.027 INF Parsing BlackStork files command=render
Jun 23 17:28:03.028 INF Loading plugin resolver command=render includeRemote=false
Jun 23 17:28:03.028 INF Loading plugin runner command=render
Jun 23 17:28:03.056 INF Publishing document command=render target=greeting
Jun 23 17:28:03.056 INF Loading document command=render target=greeting
Jun 23 17:28:04.213 INF Writing to a file command=render path=/tmp/greeting-2024_06_23.html
$
```

The engine writes the formatted HTML to the specified path:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>The Greeting</title>
</head>
<body>
 <h1 id="the-greeting">The Greeting</h1>
<p>Hello, BlackStork!</p>
<p>There are 8 planets and 146 moons in our solar system.</p>
<p>Neptune is the eighth and most distant planet in our solar system, located about 4.5 billion kilometers away from the Sun.</p>

</body>
</html>
```

To embed custom JS and CSS into the HTML document, refer to the [HTML formatter]({{< ref "plugins/builtin/formatters/html" >}}) documentation.

# Next steps

- Review the [Language Specification]({{< ref "language" >}}) for detailed syntax rules and data structures.
- Explore the [Plugin Registry]({{< ref "plugins" >}}) to see available data and content integrations.
- Browse the [Community Templates](https://github.com/blackstork-io/blackstork-templates) for production-ready reporting blocks.
- Join the [Community Slack](https://blackstork-community.slack.com/) for engineering support.
