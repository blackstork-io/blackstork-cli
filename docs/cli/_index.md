---
title: BlackStork CLI
description: Overview of blackstork-cli, the headless execution engine for rendering BlackStork templates locally and in CI/CD pipelines.
type: docs
weight: 60
---

# BlackStork CLI

The `blackstork-cli` is a source-available, headless execution engine for BlackStork templates. It is the exact same engine that powers the document generation within the managed BlackStork SaaS platform, packaged as a standalone binary for local environments.

When executed, the CLI parses your `.blackstork` configuration files, downloads any required plugins from the BlackStork registry, authenticates with your configured data sources, fetches the necessary structured data, and renders your documents locally.

Because it runs entirely on your infrastructure, `blackstork-cli` is ideal for users who need to keep sensitive security data strictly on-premise, or for engineering teams integrating automated reporting directly into existing CI/CD pipelines.

## License

`blackstork-cli` is source-available and licensed under the Business Source License (BUSL) 1.1. It is free for individual, academic, and internal business use. You may not use the CLI to provide a commercial competitive service or embed it into a proprietary commercial product. See the [License]({{< ref "license.md" >}}) page for full details.

## CLI Documentation

- [Install]({{< ref "install.md" >}}) — Download and install the `blackstork-cli` binary for your operating system.
- [CLI Reference]({{< ref "cli-reference.md" >}}) — Detailed documentation on available commands, flags, and environment variables.
- [License]({{< ref "license.md" >}}) — BlackStork CLI BUSL 1.1 license.

***

### Page 2: License Page (`cli/license.md`)

**HTML Page Title:**
`<title>License | BlackStork CLI | BlackStork</title>`

**HTML Meta Description:**
`<meta name="description" content="License information for blackstork-cli. Read the details of our Business Source License (BUSL) 1.1 and permitted use cases.">`

---

# License

The `blackstork-cli` is source-available and licensed under the **Business Source License (BUSL) 1.1**. 

We chose this license to strike a balance: we want the security community, researchers, and engineers to be able to inspect our code, build custom plugins, and use the execution engine in their own internal environments without restriction. At the same time, we need to protect our business from companies attempting to take our engine, host it, and sell it as a competing commercial service.

## Plain English Summary

**What you CAN do:**
*   Download, compile, and run `blackstork-cli` locally or in your organization's internal CI/CD pipelines.
*   Use it to generate security reports, compliance documents, and operational summaries for your internal business needs.
*   Write and execute custom plugins for your own proprietary internal tools.
*   Inspect the source code for security audits or academic research.

**What you CANNOT do:**
*   Offer `blackstork-cli` (or a modified version of it) as a hosted commercial SaaS product.
*   Embed the `blackstork-cli` engine into a proprietary commercial software product that you sell to third parties.
*   Use the software in a way that directly competes with the BlackStork SaaS platform.

If you require hosted reporting, collaborative editing, or stakeholder analytics, please use the commercial **BlackStork SaaS platform**.

## Business Source License 1.1 Boilerplate

*Note: Following the standard BUSL 1.1 structure, the code will transition to an Open Source license (Apache 2.0) after the Change Date.*

**Parameters:**
*   **Licensor:** BlackStork
*   **Licensed Work:** `blackstork-cli`
*   **Additional Use Grant:** You may make use of the Licensed Work for internal development, testing, and production generation of internal documents, provided that such use does not include offering the Licensed Work to third parties as a hosted or managed service.
*   **Change Date:** [Insert Date, e.g., 4 years from release date]
*   **Change License:** Apache License, Version 2.0

**License Text:**

Licensor hereby grants you the right to copy, modify, create derivative works, redistribute, and make non-production use of the Licensed Work. The Licensor may make an Additional Use Grant, above, permitting limited production use. 

Licensor grants you the right to copy, modify, create derivative works, redistribute and make production use of the Licensed Work, provided such use does not compete with Licensor’s products or services.

Any copy, modification, derivative work, or redistribution of the Licensed Work must include this License.

Effective on the Change Date, or the fourth anniversary of the first publicly available distribution of a specific version of the Licensed Work under this License, whichever comes first, the Licensor hereby grants you rights under the terms of the Change License, and the rights granted in the paragraph above terminate.

If your use of the Licensed Work does not comply with the requirements of this License, you must purchase a commercial license from the Licensor, its affiliated entities, or authorized resellers, or you must refrain from using the Licensed Work.
