---
title: "`eclecticiq_entities` data source"
plugin:
  name: blackstork/eclecticiq
  description: "Retrieves EclecticIQ Intelligence Center entities by ID or Lucene query. Results can include related entities of selected types and associated observables"
  tags: []
  version: "v1.0.0-rc1"
  source_github: "https://github.com/blackstork-io/blackstork-cli/tree/main/plugins/eclecticiq/"
resource:
  type: data-source
type: docs
---

{{< breadcrumbs 2 >}}

{{< plugin-resource-header "blackstork/eclecticiq" "eclecticiq" "v1.0.0-rc1" "eclecticiq_entities" "data source" >}}

## Description
Retrieves EclecticIQ Intelligence Center entities by ID or Lucene query. Results can include related entities of selected types and associated observables.

## Installation

{{< hint note >}}
**BlackStork SaaS:** Plugin dependencies are resolved automatically by the platform. You do not need to install plugins or define the `blackstork` configuration block manually.
{{< /hint >}}

To use the `eclecticiq_entities` data source locally via `blackstork-cli`, you must declare the `blackstork/eclecticiq` plugin as a dependency in your global configuration block.

```hcl
blackstork {
  plugin_versions = {
    "blackstork/eclecticiq" = ">= v1.0.0-rc1"
  }
}
```

After declaring the dependency, execute `blackstork-cli install` to fetch the plugin. See [Configuration]({{< ref "configs.md#global-configuration" >}}) for details.

## Configuration

This data source accepts the following configuration arguments within a `config data eclecticiq_entities` block:

```hcl
config data eclecticiq_entities {
  # The base URL of your EclecticIQ Platform instance.
  #
  # Required string.
  #
  # For example:
  platform_url = "https://ic-playground.eclecticiq.com"

  # The API token to authenticate with the EclecticIQ Platform. It is recommended to use environment variables to provide this value securely.
  #
  # Optional string.
  #
  # For example:
  # api_token = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9..."
  #
  # Default value:
  api_token = null
}
```

## Usage

This data source accepts the following arguments within a `data eclecticiq_entities` block:

```hcl
data eclecticiq_entities {
  # A list of STIX IDs or internal EclecticIQ UUIDs to fetch. Either 'entity_ids' or 'query' must be provided.
  #
  # Optional list of string.
  #
  # For example:
  # entity_ids = ["report--fcad1414-30b9-40ee-99f2-64c5308b9690", "814c5d00-e382-4a34-abbf-50e8937646b9"]
  #
  # Default value:
  entity_ids = null

  # A Lucene search query to find entities. This uses the same syntax as the EclecticIQ's Intelligence Center UI search. Either 'query' or 'entity_ids' must be provided.
  #
  # Optional string.
  #
  # For example:
  # query = "data.title:malware OR data.description:APT17"
  #
  # Default value:
  query = null

  # A list of STIX entity types (e.g., 'malware', 'threat-actor', 'indicator') to fetch relationships for. If set, the data source will retrieve all entities of these types connected to the matched entities.
  #
  # Optional list of string.
  #
  # For example:
  # with_related_entities_of_type = ["malware", "threat-actor", "indicator"]
  #
  # Default value:
  with_related_entities_of_type = null

  # If true, the data source will also fetch and attach all observables (extracts) associated with the matched entities.
  #
  # Optional bool.
  #
  # For example:
  # with_observables = true
  #
  # Default value:
  with_observables = false

  # Maximum number of entities to return per request. Note that the EclecticIQ API enforces a hard cap of 1000 items per query.
  #
  # Optional number.
  # Must be >= 0
  #
  # For example:
  # limit = 100
  #
  # Default value:
  limit = 1000
}
```