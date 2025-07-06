package kbclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

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

func (s *ClientTestSuite) TestAuth() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("ApiKey test_token", r.Header.Get("Authorization"))
	}, String("test_token"))
	defer srv.Close()
	client.ListSecurityCases(s.ctx, &ListSecurityCasesReq{})
}

func (s *ClientTestSuite) queryList(q url.Values, key string) []string {
	list, ok := q[key]
	s.Require().True(ok)
	return list
}

func (s *ClientTestSuite) TestWithSpaceID() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/s/space-id-123/api/cases/_find", r.URL.Path)
		s.Equal(http.MethodGet, r.Method)
		w.Write([]byte(`{
			"cases": [
				{
					"any": "data"
				}
			]
		}`))
	}, nil)
	defer srv.Close()
	req := ListSecurityCasesReq{
		SpaceID: String("space-id-123"),
	}
	result, err := client.ListSecurityCases(s.ctx, &req)
	s.NoError(err)
	s.Equal(&ListSecurityCasesRes{
		Cases: []any{
			map[string]any{
				"any": "data",
			},
		},
	}, result)
}

func (s *ClientTestSuite) TestWithoutSpaceID() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/api/cases/_find", r.URL.Path)
		s.Equal(http.MethodGet, r.Method)
		w.Write([]byte(`{
			"cases": [
				{
					"any": "data"
				}
			]
		}`))
	}, nil)
	defer srv.Close()
	req := ListSecurityCasesReq{}
	result, err := client.ListSecurityCases(s.ctx, &req)
	s.NoError(err)
	s.Equal(&ListSecurityCasesRes{
		Cases: []any{
			map[string]any{
				"any": "data",
			},
		},
	}, result)
}

func (s *ClientTestSuite) TestFull() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/s/space-id-123/api/cases/_find", r.URL.Path)
		s.Equal(http.MethodGet, r.Method)
		s.Equal([]string{"test_assignee_1", "test_assignee_2"}, s.queryList(r.URL.Query(), "assignees"))
		s.Equal("test_status", r.URL.Query().Get("status"))
		s.Equal([]string{"test_tag_1", "test_tag_2"}, s.queryList(r.URL.Query(), "tags"))
		s.Equal("test_to", r.URL.Query().Get("to"))
		s.Equal("test_search", r.URL.Query().Get("search"))
		s.Equal("test_severity", r.URL.Query().Get("severity"))
		s.Equal("test_sort_field", r.URL.Query().Get("sortField"))
		s.Equal("test_sort_order", r.URL.Query().Get("sortOrder"))
		s.Equal("test_default_search_operator", r.URL.Query().Get("defaultSearchOperator"))
		s.Equal([]string{"test_search_field_1", "test_search_field_2"}, s.queryList(r.URL.Query(), "searchFields"))
		s.Equal("test_from", r.URL.Query().Get("from"))
		s.Equal([]string{"test_owner_1", "test_owner_2"}, s.queryList(r.URL.Query(), "owner"))
		s.Equal([]string{"test_reporter_1", "test_reporter_2"}, s.queryList(r.URL.Query(), "reporters"))
		w.Write([]byte(`{
			"page": 1,
			"total": 2,
			"per_page": 3,
			"cases": [
				{
					"any": "data"
				}
			]
		}`))
	}, nil)
	defer srv.Close()
	req := ListSecurityCasesReq{
		Page:                  1,
		PerPage:               3,
		SpaceID:               String("space-id-123"),
		Assignees:             []string{"test_assignee_1", "test_assignee_2"},
		Status:                String("test_status"),
		Tags:                  []string{"test_tag_1", "test_tag_2"},
		To:                    String("test_to"),
		Search:                String("test_search"),
		Severity:              String("test_severity"),
		SortField:             String("test_sort_field"),
		SortOrder:             String("test_sort_order"),
		DefaultSearchOperator: String("test_default_search_operator"),
		SearchFields:          []string{"test_search_field_1", "test_search_field_2"},
		From:                  String("test_from"),
		Owner:                 []string{"test_owner_1", "test_owner_2"},
		Reporters:             []string{"test_reporter_1", "test_reporter_2"},
	}
	result, err := client.ListSecurityCases(s.ctx, &req)
	s.NoError(err)
	s.Equal(&ListSecurityCasesRes{
		Page:    1,
		Total:   2,
		PerPage: 3,
		Cases: []any{
			map[string]any{
				"any": "data",
			},
		},
	}, result)
}
