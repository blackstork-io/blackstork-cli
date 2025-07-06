package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type Client interface {
	SearchIssues(ctx context.Context, req *SearchIssuesReq) (*SearchIssuesRes, error)
}

type client struct {
	apiURL       string
	apiToken     string
	accountEmail string
}

func (c *client) auth(r *http.Request) {
	r.SetBasicAuth(c.accountEmail, c.apiToken)
}

func (c *client) makeHTTPClient() *http.Client {
	httpClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	return httpClient
}

func (c *client) makeURL(path ...string) (*url.URL, error) {
	addr, err := url.JoinPath(c.apiURL, path...)
	if err != nil {
		return nil, err
	}
	return url.Parse(addr)
}

func New(apiURL, accountEmail, apiToken string) Client {
	return &client{
		apiURL:       apiURL,
		accountEmail: accountEmail,
		apiToken:     apiToken,
	}
}

func (c *client) handleError(res *http.Response) error {
	var clientErr Error

	if err := json.NewDecoder(res.Body).Decode(&clientErr); err != nil {
		return err
	}

	return &clientErr
}

func (c *client) SearchIssues(ctx context.Context, req *SearchIssuesReq) (*SearchIssuesRes, error) {
	u, err := c.makeURL("/rest/api/3/search/jql")
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	c.auth(r)

	client := c.makeHTTPClient()
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, c.handleError(res)
	}

	var data SearchIssuesRes
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}
