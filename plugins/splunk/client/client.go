// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

// Package client implements the Splunk API client.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-querystring/query"
)

type Client interface {
	CreateSearchJob(ctx context.Context, req *CreateSearchJobReq) (*CreateSearchJobRes, error)
	GetSearchJobByID(ctx context.Context, req *GetSearchJobByIDReq) (*GetSearchJobByIDRes, error)
	GetSearchJobResults(ctx context.Context, req *GetSearchJobResultsReq) (*GetSearchJobResultsRes, error)
}

type client struct {
	token string
	url   string
}

func New(token, host, deployment string) Client {
	url := "https://" + host + ":8089"
	if deployment != "" {
		url = "https://" + deployment + ".splunkcloud.com:8089"
	}
	return &client{
		token: token,
		url:   url,
	}
}

func (c *client) auth(r *http.Request) {
	r.Header.Add("Authorization", "Bearer "+c.token)
}

func (c *client) CreateSearchJob(ctx context.Context, req *CreateSearchJobReq) (*CreateSearchJobRes, error) {
	v, err := query.Values(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/services/search/jobs", strings.NewReader(v.Encode()))
	if err != nil {
		return nil, err
	}
	c.auth(r)
	r.Header.Add("Accept", "application/json")
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	client := http.Client{}
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("splunk client returned status code: %d", res.StatusCode)
	}

	var data CreateSearchJobRes
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *client) GetSearchJobByID(ctx context.Context, req *GetSearchJobByIDReq) (*GetSearchJobByIDRes, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/services/search/jobs/"+req.ID, nil)
	if err != nil {
		return nil, err
	}
	c.auth(r)
	r.Header.Add("Accept", "application/json")
	client := http.Client{}
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("splunk client returned status code: %d", res.StatusCode)
	}
	var data GetSearchJobByIDRes
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *client) GetSearchJobResults(ctx context.Context, req *GetSearchJobResultsReq) (*GetSearchJobResultsRes, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/services/search/v2/jobs/"+req.ID+"/results", nil)
	if err != nil {
		return nil, err
	}
	q, err := query.Values(req)
	if err != nil {
		return nil, err
	}
	r.URL.RawQuery = q.Encode()
	c.auth(r)
	r.Header.Add("Accept", "application/json")
	client := http.Client{}
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("splunk client returned status code: %d", res.StatusCode)
	}
	var data GetSearchJobResultsRes
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}
