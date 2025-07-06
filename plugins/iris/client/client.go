package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-querystring/query"
)

type Client interface {
	ListCases(ctx context.Context, req *ListCasesReq) (*ListCasesRes, error)
	ListAlerts(ctx context.Context, req *ListAlertsReq) (*ListAlertsRes, error)
}

type client struct {
	apiURL   string
	apiKey   string
	insecure bool
}

func (c *client) auth(r *http.Request) {
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
}

func (c *client) makeHTTPClient() *http.Client {
	httpClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	if c.insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec,G402
			},
		}
	}

	return httpClient
}

func New(url, apiKey string, insecure bool) Client {
	return &client{
		apiURL:   url,
		apiKey:   apiKey,
		insecure: insecure,
	}
}

func (c *client) handleError(res *http.Response) error {
	var clientErr Error

	if err := json.NewDecoder(res.Body).Decode(&clientErr); err != nil {
		return err
	}

	return &clientErr
}

func (c *client) ListCases(ctx context.Context, req *ListCasesReq) (*ListCasesRes, error) {
	u, err := url.Parse(c.apiURL + "/manage/cases/filter")
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

	var data ListCasesRes
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

func (c *client) ListAlerts(ctx context.Context, req *ListAlertsReq) (*ListAlertsRes, error) {
	u, err := url.Parse(c.apiURL + "/alerts/filter")
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

	var data ListAlertsRes
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}
