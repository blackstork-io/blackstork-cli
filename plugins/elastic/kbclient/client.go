package kbclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-querystring/query"
)

func String(s string) *string {
	return &s
}

func Bool(b bool) *bool {
	return &b
}

func Int(i int) *int {
	return &i
}

type ListSecurityCasesReq struct {
	SpaceID               *string  `url:"-"`
	Assignees             []string `url:"assignees,omitempty"`
	DefaultSearchOperator *string  `url:"defaultSearchOperator,omitempty"`
	From                  *string  `url:"from,omitempty"`
	Owner                 []string `url:"owner,omitempty"`
	Page                  int      `url:"page"`
	PerPage               int      `url:"perPage"`
	Reporters             []string `url:"reporters,omitempty"`
	Search                *string  `url:"search,omitempty"`
	SearchFields          []string `url:"searchFields,omitempty"`
	Severity              *string  `url:"severity,omitempty"`
	SortField             *string  `url:"sortField,omitempty"`
	SortOrder             *string  `url:"sortOrder,omitempty"`
	Status                *string  `url:"status,omitempty"`
	Tags                  []string `url:"tags,omitempty"`
	To                    *string  `url:"to,omitempty"`
}

type ListSecurityCasesRes struct {
	Page    int   `json:"page"`
	Total   int   `json:"total"`
	PerPage int   `json:"per_page"`
	Cases   []any `json:"cases"`
}

type Client interface {
	ListSecurityCases(ctx context.Context, req *ListSecurityCasesReq) (*ListSecurityCasesRes, error)
}

type client struct {
	url    string
	apiKey *string
}

func New(url string, apiKey *string) Client {
	return &client{
		url:    url,
		apiKey: apiKey,
	}
}

func (c *client) auth(r *http.Request) {
	if c.apiKey != nil {
		r.Header.Set("Authorization", "ApiKey "+*c.apiKey)
	}
}

func (c *client) ListSecurityCases(ctx context.Context, req *ListSecurityCasesReq) (*ListSecurityCasesRes, error) {
	var u *url.URL
	var err error
	if req.SpaceID != nil {
		u, err = url.Parse(c.url + "/s/" + *req.SpaceID + "/api/cases/_find")
	} else {
		u, err = url.Parse(c.url + "/api/cases/_find")
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
	c.auth(r)
	client := http.Client{
		Timeout: 15 * time.Second,
	}
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kbclient client returned status code: %d", res.StatusCode)
	}
	defer res.Body.Close()
	var data ListSecurityCasesRes
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}
