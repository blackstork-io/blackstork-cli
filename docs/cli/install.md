---
title: Install
description: Download and install the blackstork-cli execution engine and resolve plugin dependencies to render templates locally.
type: docs
weight: 10
---

# Install

## Installing `blackstork-cli`

### Homebrew

To install `blackstork-cli` on macOS using [Homebrew](https://brew.sh/), run the following commands:

```bash
# Install the CLI from the BlackStork tap
brew install blackstork-io/tools/blackstork-cli

# Verify the installation
blackstork-cli --version
```

We recommend using the `blackstork-io/tools` tap for Homebrew installations, as it automatically updates with every official release.

### GitHub releases

Pre-compiled `blackstork-cli` binaries for Windows, macOS, and Linux are available on the project's [GitHub Releases](https://github.com/blackstork-io/blackstork-cli/releases) page.

To install manually:

1. **Download:** Select and download the release archive corresponding to your operating system and architecture.
2. **Unpack:** Extract the executable from the archive.
3. **Execute:** Add the binary to your system `$PATH` or execute it directly from the extracted directory.

For example, to install the latest macOS (arm64) release:

```bash
# Create a destination directory
mkdir blackstork-bin

# Download the latest macOS arm64 release
wget https://github.com/blackstork-io/blackstork-cli/releases/latest/download/blackstork-cli_darwin_arm64.tar.gz -O ./blackstork-cli_darwin_arm64.tar.gz

# Extract the binary
tar -xvzf ./blackstork-cli_darwin_arm64.tar.gz -C ./blackstork-bin

# Verify the installation
./blackstork-bin/blackstork-cli --help
```

## Installing plugins

The `blackstork-cli` relies on [plugins]({{< ref "../plugins/_index.md" >}}) to connect with external data sources, process content via APIs, and format documents.

Before evaluating a template that relies on external integrations, download its required plugins with the `install` command. The command queries the official BlackStork registry (`https://registry.blackstork.io`).

To resolve and install plugins:

1. **Declare dependencies:** Ensure the plugins are listed in the `plugin_versions` map within the global `blackstork` configuration block (see [Global configuration]({{< ref "configs.md#global-configuration" >}})).

  ```hcl
  blackstork {
    plugin_versions = {
      "blackstork/openai"  = ">= 0.4.0",
      "blackstork/elastic" = ">= 0.4.0",
    }
  }
  ```

2. **Run the install command:** Execute `blackstork-cli install` in the same directory as your configuration files. Add `--upgrade` to select newer versions allowed by the configured constraints.

  ```text
  $ blackstork-cli install
  Mar 11 19:20:09.085 INF Searching plugin name=blackstork/elastic constraints=">=v0.4.0"
  Mar 11 19:20:09.522 INF Installing plugin name=blackstork/elastic version=0.4.0
  Mar 11 19:20:10.769 INF Searching plugin name=blackstork/openai constraints=">=v0.4.0"
  Mar 11 19:20:10.787 INF Installing plugin name=blackstork/openai version=0.4.0
  $
  ```

The CLI downloads the plugin binaries and caches them in the `.blackstork` directory (or the custom `cache_dir` specified in your global configuration).

{{< hint note >}}
You do not need to run `blackstork-cli install` if your templates solely rely on components from the [built-in plugin]({{< ref "../plugins/builtin/_index.md" >}}). The built-in components ship directly inside the CLI binary.
{{</ hint >}}

## Next steps

See the [CLI Reference]({{< ref "cli-reference.md" >}}) to learn how to execute template builds and configure local logging.
