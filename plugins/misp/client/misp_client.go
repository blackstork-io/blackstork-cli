// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

// Package client implements the MISP API client.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	baseURL string
	apiKey  string

	client *http.Client
}

type ClientOption func(*Client)

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.client = httpClient
	}
}

func NewClient(baseURL, apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.client == nil {
		c.client = &http.Client{}
	}
	return c
}

func (client *Client) auth(r *http.Request) {
	r.Header.Set("Authorization", client.apiKey)
}

func (client *Client) Do(ctx context.Context, method, path string, payload interface{}) (resp *http.Response, err error) {
	var body io.Reader
	if payload != nil {
		jsonBuf, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		reader := bytes.NewReader(jsonBuf)
		body = io.NopCloser(reader)
	}

	req, err := http.NewRequest(method, client.baseURL+path, body)
	if err != nil {
		return resp, err
	}
	req = req.WithContext(ctx)

	req.Header = make(http.Header)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	client.auth(req)
	resp, err = client.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("MISP server replied status=%d", resp.StatusCode)
	}

	return resp, nil
}

func (client *Client) RestSearchEvents(ctx context.Context, req RestSearchEventsRequest) (events RestSearchEventsResponse, err error) {
	resp, err := client.Do(ctx, http.MethodPost, "/events/restSearch", req)
	if err != nil {
		return events, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&events)
	if err != nil {
		return events, err
	}
	return events, err
}

func (client *Client) AddEventReport(ctx context.Context, req AddEventReportRequest) (events AddEventReportResponse, err error) {
	resp, err := client.Do(ctx, http.MethodPost, "/event_reports/add/"+req.EventID, req)
	if err != nil {
		return events, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&events)
	if err != nil {
		return events, err
	}
	return events, err
}
