// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package eclecticiq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugins/eclecticiq/client"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

const (
	relatedEntitiesLimit = 100
)

type ExtendedEntity struct {
	*client.Entity

	// Map of related entities by their type
	RelatedEntities map[string][]*client.Entity `json:"related_entities,omitempty"`
	Observables     []*client.Extract           `json:"observables,omitempty"`
}

func makeEIQEntitiesDataSource(log *slog.Logger, loader EIQClientLoaderFn) *plugin.DataSource {
	return &plugin.DataSource{
		DataFunc: fetchEIQEntities(log, loader),
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "platform_url",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
					Doc:         `The base URL of your EclecticIQ Platform instance.`,
					ExampleVal:  cty.StringVal("https://ic-playground.eclecticiq.com"),
				},
				{
					Name:        "api_token",
					Type:        cty.String,
					Constraints: constraint.NonNull,
					Secret:      true,
					Doc:         `The API token to authenticate with the EclecticIQ Platform. It is recommended to use environment variables to provide this value securely.`,
					ExampleVal:  cty.StringVal("eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9..."),
				},
			},
		},
		Args: &dataspec.RootSpec{
			Required: true,
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "entity_ids",
					Type: cty.List(cty.String),
					Doc:  `A list of STIX IDs or internal EclecticIQ UUIDs to fetch. Either 'entity_ids' or 'query' must be provided.`,
					ExampleVal: cty.ListVal([]cty.Value{
						cty.StringVal("report--fcad1414-30b9-40ee-99f2-64c5308b9690"),
						cty.StringVal("814c5d00-e382-4a34-abbf-50e8937646b9"),
					}),
				},
				{
					Name:       "query",
					Type:       cty.String,
					Doc:        `A Lucene search query to find entities. This uses the same syntax as the EclecticIQ's Intelligence Center UI search. Either 'query' or 'entity_ids' must be provided.`,
					ExampleVal: cty.StringVal("data.title:malware OR data.description:APT17"),
				},
				{
					Name: "with_related_entities_of_type",
					Type: cty.List(cty.String),
					Doc:  `A list of STIX entity types (e.g., 'malware', 'threat-actor', 'indicator') to fetch relationships for. If set, the data source will retrieve all entities of these types connected to the matched entities.`,
					ExampleVal: cty.ListVal([]cty.Value{
						cty.StringVal("malware"),
						cty.StringVal("threat-actor"),
						cty.StringVal("indicator"),
					}),
				},
				{
					Name:       "with_observables",
					Type:       cty.Bool,
					DefaultVal: cty.BoolVal(false),
					Doc:        `If true, the data source will also fetch and attach all observables (extracts) associated with the matched entities.`,
					ExampleVal: cty.BoolVal(true),
				},
				{
					Name:         "limit",
					Type:         cty.Number,
					Constraints:  constraint.NonNull,
					DefaultVal:   cty.NumberIntVal(1000),
					MinInclusive: cty.NumberIntVal(0),
					Doc:          `Maximum number of entities to return per request. Note that the EclecticIQ API enforces a hard cap of 1000 items per query.`,
					ExampleVal:   cty.NumberIntVal(100),
				},
			},
		},
	}
}

func fetchEIQEntities(log *slog.Logger, loader EIQClientLoaderFn) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		url := params.Config.GetAttrVal("platform_url").AsString()
		token := params.Config.GetAttrVal("api_token").AsString()

		var entityIDs []string
		if entityIDsVals := params.Args.GetAttrVal("entity_ids"); !entityIDsVals.IsNull() {
			entityIDs = toStringSlice(entityIDsVals)
		}

		var query *string
		if queryVal := params.Args.GetAttrVal("query"); !queryVal.IsNull() {
			query = new(queryVal.AsString())
		}

		fetchObs := params.Args.GetAttrVal("with_observables").True()

		var fetchRelatedTypes []string
		if relatedTypes := params.Args.GetAttrVal("with_related_entities_of_type"); !relatedTypes.IsNull() {
			fetchRelatedTypes = toStringSlice(relatedTypes)
		}

		limit, _ := params.Args.GetAttrVal("limit").AsBigFloat().Int64()

		if len(entityIDs) == 0 && query == nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Either `entity_ids` or `query` must be set",
			}}
		}

		eiqClient, err := loader(url, token)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to create EclecticIQ Platform client",
				Detail:   err.Error(),
			}}
		}

		var entities []*client.Entity

		if len(entityIDs) > 0 {
			log.DebugContext(ctx, "Fetching entities by ID", "entity_ids", entityIDs)
			entities, err = eiqClient.FetchEntitiesByID(ctx, entityIDs)
		} else if query != nil {
			log.DebugContext(ctx, "Querying entities", "query", *query, "limit", limit)
			entities, err = eiqClient.QueryEntities(ctx, *query, int(limit))
		}

		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to query entities",
				Detail:   err.Error(),
			}}
		}

		log.DebugContext(ctx, "Entities retrieved", "entities_count", len(entities))

		extendedEntities := utils.FnMap(entities, func(entity *client.Entity) *ExtendedEntity {
			ext := &ExtendedEntity{
				Entity: entity,
			}

			_log := log.With("entity_id", entity.StixID)

			// fetch observables if requested
			if fetchObs {
				//_log.DebugContext(ctx, "Fetching observables")
				obs, err := eiqClient.FetchObservables(ctx, entity.StixID)
				if err != nil {
					_log.WarnContext(ctx, "Failed to fetch observables", "err", err)
				} else {
					ext.Observables = obs
					//_log.DebugContext(ctx, "Observables retrieved", "obs_count", len(obs))
				}
			}

			// fetch related entities + relationship types & directions
			if len(fetchRelatedTypes) > 0 {
				//_log.DebugContext(ctx, "Fetching related entities", "related_types", fetchRelatedTypes)
				relatedEntities, err := eiqClient.FetchRelatedEntities(
					ctx,
					entity.InternalID,
					fetchRelatedTypes,
					relatedEntitiesLimit,
				)
				if err != nil {
					_log.WarnContext(ctx, "Failed to fetch related entities", "err", err)
				} else {
					//_log.DebugContext(ctx, "Related entities retrieved", "related_entities_count", len(relatedEntities))

					ext.RelatedEntities = make(map[string][]*client.Entity)

					for _, e := range relatedEntities {
						ext.RelatedEntities[e.Type] = append(ext.RelatedEntities[e.Type], e)
					}
				}

				// rels, err := eiqClient.FetchRelationships(ctx, entity.StixID, 100)
				// _log.DebugContext(ctx, "JUST A CHECK: relationships found", "rels_count", len(rels), "rels", rels, "err", err)
			}

			return ext
		})

		encoded, err := toPlugindata(extendedEntities)
		if err != nil {
			log.ErrorContext(ctx, "Error while encoding entities", "err", err)
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to encode received entities",
				Detail:   err.Error(),
			}}
		}

		return encoded, nil
	}
}

func toStringSlice(val cty.Value) []string {
	list := make([]string, 0, val.LengthInt())
	iter := val.ElementIterator()
	for iter.Next() {
		_, v := iter.Element()
		list = append(list, v.AsString())
	}
	return list
}

func toPlugindata(data any) (plugindata.Data, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert data: %w", err)
	}
	return plugindata.UnmarshalJSON(raw)
}
