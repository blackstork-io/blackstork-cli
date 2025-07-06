package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
)

const (
	version    = "2024-01-23"
	defaultURL = "https://api.snyk.io/rest"
)

func String(s string) *string {
	return &s
}

func Bool(b bool) *bool {
	return &b
}

type StringList []string

func (list StringList) EncodeValues(key string, v *url.Values) error {
	if len(list) == 0 {
		return nil
	}
	v.Add(key, strings.Join(list, ","))
	return nil
}

type ListIssuesReq struct {
	GroupID                *string    `url:"-"`
	OrgID                  *string    `url:"-"`
	StartingAfter          *string    `url:"starting_after,omitempty"`
	Limit                  int        `url:"limit,omitempty"`
	ScanItemID             *string    `url:"scan_item.id,omitempty"`
	ScanItemType           *string    `url:"scan_item.type,omitempty"`
	Type                   *string    `url:"type,omitempty"`
	UpdatedBefore          *string    `url:"updated_before,omitempty"`
	UpdatedAfter           *string    `url:"updated_after,omitempty"`
	CreatedBefore          *string    `url:"created_before,omitempty"`
	CreatedAfter           *string    `url:"created_after,omitempty"`
	EffectiveSeverityLevel StringList `url:"effective_severity_level,omitempty"`
	Status                 StringList `url:"status,omitempty"`
	Ignored                *bool      `url:"ignored,omitempty"`
}

type ListIssuesRes struct {
	Data  []any  `json:"data"`
	Links *Links `json:"links"`
}

type Links struct {
	Next *string `json:"next"`
}

type Client interface {
	ListIssues(ctx context.Context, req *ListIssuesReq) (*ListIssuesRes, error)
}

type client struct {
	apiKey string
	url    string
}

func New(apiKey string) Client {
	return &client{
		apiKey: apiKey,
		url:    defaultURL,
	}
}

func (c *client) prepare(r *http.Request) {
	r.Header.Set("Authorization", "Token "+c.apiKey)
	r.Header.Set("Accept", "application/vnd.api+json")
	q := r.URL.Query()
	q.Add("version", version)
	r.URL.RawQuery = q.Encode()
}

func (c *client) ListIssues(ctx context.Context, req *ListIssuesReq) (*ListIssuesRes, error) {
	var u *url.URL
	var err error
	if req.GroupID != nil {
		u, err = url.Parse(c.url + "/groups/" + *req.GroupID + "/issues")
	} else {
		u, err = url.Parse(c.url + "/orgs/" + *req.OrgID + "/issues")
	}
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
	c.prepare(r)
	client := http.Client{
		Timeout: 15 * time.Second,
	}
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("snyk client returned status code: %d", res.StatusCode)
	}
	defer res.Body.Close()
	var data ListIssuesRes
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}
