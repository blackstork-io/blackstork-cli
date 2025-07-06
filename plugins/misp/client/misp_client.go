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
	baseUrl string
	apiKey  string

	client *http.Client
}

type ClientOption func(*Client)

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.client = httpClient
	}
}

func NewClient(baseUrl, apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		baseUrl: baseUrl,
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

func (c *Client) auth(r *http.Request) {
	r.Header.Set("Authorization", c.apiKey)
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

	req, err := http.NewRequest(method, client.baseUrl+path, body)
	if err != nil {
		return
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
		return
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&events)
	if err != nil {
		return
	}
	return
}

func (client *Client) AddEventReport(ctx context.Context, req AddEventReportRequest) (events AddEventReportResponse, err error) {
	resp, err := client.Do(ctx, http.MethodPost, "/event_reports/add/"+req.EventId, req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&events)
	if err != nil {
		return
	}
	return
}
