---
title: "`nist_nvd_cves` data source"
plugin:
  name: blackstork/nist_nvd
  description: "Retrieves CVE records from the NIST National Vulnerability Database. Supports CVE, CPE, CVSS, CWE, keyword, source, publication date, modification date, and rejected-record filters"
  tags: []
  version: "v1.0.0"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/nistnvd/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/nist_nvd" "nist_nvd" "v1.0.0" "nist_nvd_cves" "data source" >}}

## Description
Retrieves CVE records from the NIST National Vulnerability Database. Supports CVE, CPE, CVSS, CWE, keyword, source, publication date, modification date, and rejected-record filters.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `nist_nvd_cves` data source locally via `blackstork-cli`, you must declare the `blackstork/nist_nvd` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/nist_nvd" = ">= v1.0.0"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data nist_nvd_cves` block:

```hcl
config data nist_nvd_cves {
  # Optional string.
  # Default value:
  api_key = null
}
```

## Usage

This data source accepts the following arguments within a `data nist_nvd_cves` block:

```hcl
data nist_nvd_cves {
  # Optional string.
  # Default value:
  last_mod_start_date = null

  # Optional string.
  # Default value:
  last_mod_end_date = null

  # Optional string.
  # Default value:
  pub_start_date = null

  # Optional string.
  # Default value:
  pub_end_date = null

  # Optional string.
  # Default value:
  cpe_name = null

  # Optional string.
  # Default value:
  cve_id = null

  # Optional string.
  # Default value:
  cvss_v3_metrics = null

  # Optional string.
  # Default value:
  cvss_v3_severity = null

  # Optional string.
  # Default value:
  cwe_id = null

  # Optional string.
  # Default value:
  keyword_search = null

  # Optional string.
  # Default value:
  virtual_match_string = null

  # Optional string.
  # Default value:
  source_identifier = null

  # Optional bool.
  # Default value:
  has_cert_alerts = null

  # Optional bool.
  # Default value:
  has_kev = null

  # Optional bool.
  # Default value:
  has_cert_notes = null

  # Optional bool.
  # Default value:
  is_vulnerable = null

  # Optional bool.
  # Default value:
  keyword_exact_match = null

  # Optional bool.
  # Default value:
  no_rejected = null

  # Optional number.
  # Default value:
  limit = null
}
```