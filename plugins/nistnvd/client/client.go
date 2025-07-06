package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-querystring/query"
)

func String(s string) *string {
	return &s
}

func Bool(b bool) BoolValue {
	return BoolValue(b)
}

func Int(i int) *int {
	return &i
}

type BoolValue bool

func (b BoolValue) EncodeValues(key string, v *url.Values) error {
	if b {
		v.Add(key, "")
	}
	return nil
}

type ListCVESReq struct {
	ResultsPerPage     int       `url:"resultsPerPage"`
	StartIndex         int       `url:"startIndex"`
	CPEName            *string   `url:"cpeName,omitempty"`
	LastModStartDate   *string   `url:"lastModStartDate,omitempty"`
	LastModEndDate     *string   `url:"lastModEndDate,omitempty"`
	PubStartDate       *string   `url:"pubStartDate,omitempty"`
	PubEndDate         *string   `url:"pubEndDate,omitempty"`
	VirtualMatchString *string   `url:"virtualMatchString,omitempty"`
	CVEID              *string   `url:"cveId,omitempty"`
	CVSSV3Metrics      *string   `url:"cvssV3Metrics,omitempty"`
	CVSSV3Severity     *string   `url:"cvssV3Severity,omitempty"`
	CWEID              *string   `url:"cweId,omitempty"`
	HasCertAlerts      BoolValue `url:"hasCertAlerts,omitempty"`
	HasCertNotes       BoolValue `url:"hasCertNotes,omitempty"`
	HasKev             BoolValue `url:"hasKev,omitempty"`
	IsVulnerable       BoolValue `url:"isVulnerable,omitempty"`
	NoRejected         BoolValue `url:"noRejected,omitempty"`
	KeywordSearch      *string   `url:"keywordSearch,omitempty"`
	KeywordExactMatch  BoolValue `url:"keywordExactMatch,omitempty"`
	SourceIdentifier   *string   `url:"sourceIdentifier,omitempty"`
}

type ListCVESRes struct {
	ResultsPerPage  int   `json:"resultsPerPage"`
	StartIndex      int   `json:"startIndex"`
	TotalResults    int   `json:"totalResults"`
	Vulnerabilities []any `json:"vulnerabilities"`
}

type Client interface {
	ListCVES(ctx context.Context, req *ListCVESReq) (*ListCVESRes, error)
}

const (
	defaultBaseURL = "https://services.nvd.nist.gov"
)

type client struct {
	url    string
	apiKey *string
}

func (c *client) auth(r *http.Request) {
	if c.apiKey != nil {
		q := r.URL.Query()
		q.Set("apiKey", *c.apiKey)
		r.URL.RawQuery = q.Encode()
	}
}

func New(apiKey *string) Client {
	return &client{
		url:    defaultBaseURL,
		apiKey: apiKey,
	}
}

func (c *client) ListCVES(ctx context.Context, req *ListCVESReq) (*ListCVESRes, error) {
	u, err := url.Parse(c.url + "/rest/json/cves/2.0")
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
	client := http.Client{
		Timeout: 15 * time.Second,
	}
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var data ListCVESRes
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}
