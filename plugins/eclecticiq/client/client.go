// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	relationshipsLimit = 100
)

type Client interface {
	FetchEntitiesByID(ctx context.Context, entityIDs []string) ([]*Entity, error)
	QueryEntities(ctx context.Context, query string, limit int) ([]*Entity, error)
	FetchObservables(ctx context.Context, entityID string) ([]*Extract, error)
	FetchRelationships(ctx context.Context, entityID string, limit int) ([]*Relationship, error)
	FetchRelatedEntities(ctx context.Context, internalID string, relatedTypes []string, limit int) ([]*Entity, error)
	// FetchRelatedEntitiesWithRelationships(ctx context.Context, entityID string, relatedTypes []string, limit int) (map[string][]*RelatedEntity, error)
}

type client struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
}

func New(platformURL, token string) (Client, error) {
	u, err := url.Parse(platformURL)
	if err != nil {
		return nil, err
	}

	return &client{
		baseURL:    u,
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *client) doReq(
	ctx context.Context,
	method, path string,
	query url.Values,
	contentType string,
	body io.Reader,
) ([]byte, error) {
	rel, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path: %w", err)
	}

	// Attach GET query parameters if provided
	if query != nil {
		rel.RawQuery = query.Encode()
	}

	reqURL := c.baseURL.ResolveReference(rel).String()

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	} else {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBytes))
	}

	return respBytes, err
}

func (c *client) FetchEntitiesByID(ctx context.Context, entityIDs []string) ([]*Entity, error) {
	if len(entityIDs) == 0 {
		return nil, nil
	}

	var conditions []string
	for _, id := range entityIDs {
		escapedID := strings.ReplaceAll(id, `"`, `\"`)
		conditions = append(conditions, fmt.Sprintf(`data.id:"%s" OR id:"%s"`, escapedID, escapedID))
	}
	luceneQuery := strings.Join(conditions, " OR ")

	return c.QueryEntities(ctx, luceneQuery, len(entityIDs))
}

func (c *client) QueryEntities(ctx context.Context, query string, limit int) ([]*Entity, error) {
	form := url.Values{}
	form.Set("filter[_lucene_search]", query)
	if limit > 0 {
		form.Set("limit", strconv.Itoa(limit))
	}

	respBytes, err := c.doReq(
		ctx,
		http.MethodPost,
		"/api/v2/entities",
		nil,
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []rawEntity `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return nil, err
	}

	return mapRawEntities(response.Data), nil
}

func (c *client) FetchObservables(ctx context.Context, entityID string) ([]*Extract, error) {
	path := fmt.Sprintf("/api/v2/observables?filter[entities]=%s", url.QueryEscape(entityID))
	respBytes, err := c.doReq(ctx, http.MethodGet, path, nil, "", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return nil, err
	}

	var extracts []*Extract
	for _, obs := range response.Data {
		extracts = append(extracts, &Extract{
			Kind:  obs.Type,
			Value: obs.Value,
		})
	}
	return extracts, nil
}

// FetchRelatedEntities fetches entities related to a given entity, optionally filtered by target entity types
func (c *client) FetchRelatedEntities(ctx context.Context, internalID string, relatedTypes []string, limit int) ([]*Entity, error) {
	// Build the target node query (query_2) based on requested entity types to push filtering to DB
	targetQuery := "*"
	if len(relatedTypes) > 0 {
		var typeConditions []string
		for _, t := range relatedTypes {
			typeConditions = append(typeConditions, fmt.Sprintf(`data.type:"%s"`, t))
		}
		targetQuery = strings.Join(typeConditions, " OR ")
	}

	payload := map[string]any{
		"data": map[string]any{
			"query_1": map[string]any{
				"es_query": map[string]any{
					"query_string": map[string]any{
						"query": fmt.Sprintf(`id:"%s"`, internalID),
					},
				},
				"node_type": "entity",
			},
			"query_2": map[string]any{
				"es_query": map[string]any{
					"query_string": map[string]any{
						"query": targetQuery,
					},
				},
				"node_type": "entity",
			},
			"relation_query": map[string]any{
				"query_string": map[string]any{"query": "*"},
			},
			"output": "query_2",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Prepare URL query parameters for the POST request
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}

	respBytes, err := c.doReq(
		ctx,
		http.MethodPost,
		"/api/v2/entities/relational-search",
		q,
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []rawEntity `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return nil, err
	}

	return mapRawEntities(response.Data), nil
}

func (c *client) FetchRelationships(ctx context.Context, entityID string, limit int) ([]*Relationship, error) {
	var allRels []*Relationship

	// 1. Fetch Outgoing Relationships (Entity is Source)
	qOut := url.Values{}
	qOut.Set("filter[data.source]", entityID)
	if limit > 0 {
		qOut.Set("limit", strconv.Itoa(limit))
	}

	respOut, err := c.doReq(ctx, http.MethodGet, "/api/v2/relationships", qOut, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed fetching outgoing relationships: %w", err)
	}

	var resOut struct {
		Data []rawRelationship `json:"data"`
	}
	if err := json.Unmarshal(respOut, &resOut); err != nil {
		return nil, err
	}
	allRels = append(allRels, mapRawRelationships(resOut.Data)...)

	// 2. Fetch Incoming Relationships (Entity is Target)
	qIn := url.Values{}
	qIn.Set("filter[data.target]", entityID)
	if limit > 0 {
		qIn.Set("limit", strconv.Itoa(limit))
	}

	respIn, err := c.doReq(ctx, http.MethodGet, "/api/v2/relationships", qIn, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed fetching incoming relationships: %w", err)
	}

	var resIn struct {
		Data []rawRelationship `json:"data"`
	}
	if err := json.Unmarshal(respIn, &resIn); err != nil {
		return nil, err
	}
	allRels = append(allRels, mapRawRelationships(resIn.Data)...)

	// Deduplicate just in case (though edges should be distinct)
	dedup := make(map[string]*Relationship)
	for _, r := range allRels {
		dedup[r.ID] = r
	}

	var finalRels []*Relationship
	for _, r := range dedup {
		finalRels = append(finalRels, r)
	}

	return finalRels, nil
}

func (c *client) FetchRelatedEntitiesWithRelationships(ctx context.Context, entityID string, relatedTypes []string, limit int) (map[string][]*RelatedEntity, error) {
	relLimit := min(limit, relationshipsLimit)
	rels, err := c.FetchRelationships(ctx, entityID, relLimit)
	if err != nil {
		return nil, err
	}

	if len(rels) == 0 {
		return nil, nil
	}

	var relatedIDs []string

	// collect all unique IDs on the other side of the edges
	idSet := make(map[string]bool)
	for _, r := range rels {
		otherID := r.Target
		if r.Target == entityID {
			otherID = r.Source
		}
		if !idSet[otherID] {
			relatedIDs = append(relatedIDs, otherID)
			idSet[otherID] = true
		}
	}

	// fetch the related entities dynamically based on the edge count
	relatedEntities, err := c.FetchEntitiesByID(ctx, relatedIDs)
	if err != nil {
		return nil, err
	}

	relatedEntitiesMap := make(map[string][]*RelatedEntity)

	// Build a fast lookup map for the fetched entities
	entityLookup := make(map[string]*Entity)
	for _, re := range relatedEntities {
		entityLookup[re.StixID] = re
	}

	// iterate over the edges to build our related items.
	for _, r := range rels {
		otherID := r.Target
		isOutgoing := true // the main entity is the Source

		if r.Target == entityID {
			otherID = r.Source
			isOutgoing = false // the main entity is the Target
		}

		if rEntity, exists := entityLookup[otherID]; exists {
			if slices.Contains(relatedTypes, rEntity.Type) {
				relatedEntitiesMap[rEntity.Type] = append(relatedEntitiesMap[rEntity.Type], &RelatedEntity{
					Entity:       rEntity,
					RelationType: r.Key,
					IsOutgoing:   isOutgoing,
				})
			}
		}
	}
	return relatedEntitiesMap, nil
}
