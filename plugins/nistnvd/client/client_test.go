package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
}

func (s *ClientTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
}

func (s *ClientTestSuite) TearDownTest() {
	s.cancel()
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) mock(fn http.HandlerFunc, apiKey *string) (Client, *httptest.Server) {
	srv := httptest.NewServer(fn)
	cli := &client{
		url:    srv.URL,
		apiKey: apiKey,
	}
	return cli, srv
}

func hasQueryKey(q *http.Request, key string) bool {
	_, ok := q.URL.Query()[key]
	return ok
}

func (s *ClientTestSuite) TestAuth() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("api_key", r.URL.Query().Get("apiKey"))
	}, String("api_key"))
	defer srv.Close()
	client.ListCVES(s.ctx, &ListCVESReq{})
}

func (s *ClientTestSuite) TestListCVES() {
	ts := time.Unix(123, 0).UTC().Format(time.RFC3339)
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/rest/json/cves/2.0", r.URL.Path)
		s.Equal(http.MethodGet, r.Method)
		s.Equal("10", r.URL.Query().Get("resultsPerPage"))
		s.Equal("1", r.URL.Query().Get("startIndex"))
		s.Equal("cpe:2.3:o:microsoft:windows_10:1607", r.URL.Query().Get("cpeName"))
		s.True(hasQueryKey(r, "isVulnerable"))
		s.Equal(ts, r.URL.Query().Get("lastModStartDate"))
		s.Equal(ts, r.URL.Query().Get("lastModEndDate"))
		s.Equal(ts, r.URL.Query().Get("pubStartDate"))
		s.Equal(ts, r.URL.Query().Get("pubEndDate"))
		s.Equal("virtual", r.URL.Query().Get("virtualMatchString"))
		s.Equal("cve-2021-1234", r.URL.Query().Get("cveId"))
		s.Equal("cvssv3", r.URL.Query().Get("cvssV3Metrics"))
		s.Equal("high", r.URL.Query().Get("cvssV3Severity"))
		s.Equal("cwe-123", r.URL.Query().Get("cweId"))
		s.True(hasQueryKey(r, "hasCertAlerts"))
		s.True(hasQueryKey(r, "hasCertNotes"))
		s.True(hasQueryKey(r, "hasKev"))
		s.True(hasQueryKey(r, "noRejected"))
		w.Write([]byte(`{
			"resultsPerPage": 10,
			"startIndex": 1,
			"totalResults": 1,
			"vulnerabilities": [
				{
					"any": "data"
				}
			]
		}`))
	}, nil)
	defer srv.Close()
	req := ListCVESReq{
		ResultsPerPage:     10,
		StartIndex:         1,
		CPEName:            String("cpe:2.3:o:microsoft:windows_10:1607"),
		IsVulnerable:       Bool(true),
		LastModStartDate:   String(ts),
		LastModEndDate:     String(ts),
		PubStartDate:       String(ts),
		PubEndDate:         String(ts),
		VirtualMatchString: String("virtual"),
		CVEID:              String("cve-2021-1234"),
		CVSSV3Metrics:      String("cvssv3"),
		CVSSV3Severity:     String("high"),
		CWEID:              String("cwe-123"),
		HasCertAlerts:      Bool(true),
		HasCertNotes:       Bool(true),
		HasKev:             Bool(true),
		NoRejected:         Bool(true),
		KeywordSearch:      String("keyword"),
		KeywordExactMatch:  Bool(true),
		SourceIdentifier:   String("source"),
	}
	result, err := client.ListCVES(s.ctx, &req)
	s.NoError(err)
	s.Equal(&ListCVESRes{
		ResultsPerPage: 10,
		StartIndex:     1,
		TotalResults:   1,
		Vulnerabilities: []any{
			map[string]any{
				"any": "data",
			},
		},
	}, result)
}

func (s *ClientTestSuite) TestListCVESError() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}, nil)
	defer srv.Close()
	req := ListCVESReq{}
	_, err := client.ListCVES(s.ctx, &req)
	s.Error(err)
}

func (s *ClientTestSuite) TestDefaultClientURL() {
	cli := New(nil)
	s.Equal("https://services.nvd.nist.gov", cli.(*client).url)
}
