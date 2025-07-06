package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-querystring/query"
)

var defaultAPIBaseURL = "https://www.virustotal.com/api/v3"

type Client interface {
	GetUserAPIUsage(ctx context.Context, req *GetUserAPIUsageReq) (*GetUserAPIUsageRes, error)
	GetGroupAPIUsage(ctx context.Context, req *GetGroupAPIUsageReq) (*GetGroupAPIUsageRes, error)
}

type client struct {
	url string
	key string
}

func New(key string) Client {
	return &client{
		url: defaultAPIBaseURL,
		key: key,
	}
}

func (c *client) auth(r *http.Request) {
	r.Header.Set("x-apikey", c.key)
}

func (c *client) GetUserAPIUsage(ctx context.Context, req *GetUserAPIUsageReq) (*GetUserAPIUsageRes, error) {
	u, err := url.Parse(c.url + "/users/" + req.User + "/api_usage")
	if err != nil {
		return nil, err
	}
	q, err := query.Values(req)
	if err != nil {
		return nil, err
	}
	u.RawQuery = q.Encode()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	c.auth(r)
	client := http.Client{
		Timeout: 15 * time.Second,
	}
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		var data ErrorRes
		if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
			return nil, err
		}
		return nil, data.Error
	}
	defer res.Body.Close()
	var data GetUserAPIUsageRes
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *client) GetGroupAPIUsage(ctx context.Context, req *GetGroupAPIUsageReq) (*GetGroupAPIUsageRes, error) {
	u, err := url.Parse(c.url + "/groups/" + req.Group + "/api_usage")
	if err != nil {
		return nil, err
	}
	q, err := query.Values(req)
	if err != nil {
		return nil, err
	}
	u.RawQuery = q.Encode()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	c.auth(r)
	client := http.Client{
		Timeout: 15 * time.Second,
	}
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		var data ErrorRes
		if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
			return nil, err
		}
		return nil, data.Error
	}
	defer res.Body.Close()
	var data GetGroupAPIUsageRes
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}
