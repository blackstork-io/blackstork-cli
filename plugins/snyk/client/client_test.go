package client

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func (s *ClientTestSuite) mock(fn http.HandlerFunc, apiKey string) (Client, *httptest.Server) {
	srv := httptest.NewServer(fn)
	cli := &client{
		url:    srv.URL,
		apiKey: apiKey,
	}
	return cli, srv
}

func (s *ClientTestSuite) TestPrepare() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("Token test_token", r.Header.Get("Authorization"))
		s.Equal("application/vnd.api+json", r.Header.Get("Accept"))
		s.Equal(version, r.URL.Query().Get("version"))
	}, "test_token")
	defer srv.Close()
	client.ListIssues(s.ctx, &ListIssuesReq{
		GroupID: String("test_group_id"),
	})
}

func (s *ClientTestSuite) TestWithGroupID() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/groups/test_group_id/issues", r.URL.Path)
		s.Equal(http.MethodGet, r.Method)
		w.Write([]byte(`{
			"data": [
				{
					"any": "data"
				}
			],
			"links": {
				"next": "test_next"
			}
		}`))
	}, "test_token")
	defer srv.Close()
	req := ListIssuesReq{
		GroupID: String("test_group_id"),
	}
	result, err := client.ListIssues(s.ctx, &req)
	s.NoError(err)
	s.Equal(&ListIssuesRes{
		Data: []any{
			map[string]any{
				"any": "data",
			},
		},
		Links: &Links{
			Next: String("test_next"),
		},
	}, result)
}

func (s *ClientTestSuite) TestWithOrgID() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/orgs/test_org_id/issues", r.URL.Path)
		s.Equal(http.MethodGet, r.Method)
		w.Write([]byte(`{
			"data": [
				{
					"any": "data"
				}
			],
			"links": {
				"next": "test_next"
			}
		}`))
	}, "test_token")
	defer srv.Close()
	req := ListIssuesReq{
		OrgID: String("test_org_id"),
	}
	result, err := client.ListIssues(s.ctx, &req)
	s.NoError(err)
	s.Equal(&ListIssuesRes{
		Data: []any{
			map[string]any{
				"any": "data",
			},
		},
		Links: &Links{
			Next: String("test_next"),
		},
	}, result)
}

func (s *ClientTestSuite) TestFull() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/groups/test_group_id/issues", r.URL.Path)
		s.Equal(http.MethodGet, r.Method)
		s.Equal("test_starting_after", r.URL.Query().Get("starting_after"))
		s.Equal("test_scan_item_id", r.URL.Query().Get("scan_item.id"))
		s.Equal("test_scan_item_type", r.URL.Query().Get("scan_item.type"))
		s.Equal("test_type", r.URL.Query().Get("type"))
		s.Equal("test_updated_before", r.URL.Query().Get("updated_before"))
		s.Equal("test_updated_after", r.URL.Query().Get("updated_after"))
		s.Equal("test_created_before", r.URL.Query().Get("created_before"))
		s.Equal("test_created_after", r.URL.Query().Get("created_after"))
		s.Equal("test_effective_severity_level_1,test_effective_severity_level_2", r.URL.Query().Get("effective_severity_level"))
		s.Equal("test_status_1,test_status_2", r.URL.Query().Get("status"))
		s.Equal("true", r.URL.Query().Get("ignored"))
		s.Equal("10", r.URL.Query().Get("limit"))
		w.Write([]byte(`{
			"data": [
				{
					"any": "data"
				}
			],
			"links": {
				"next": "test_next"
			}
		}`))
	}, "test_token")
	defer srv.Close()
	req := ListIssuesReq{
		GroupID:                String("test_group_id"),
		Limit:                  10,
		StartingAfter:          String("test_starting_after"),
		ScanItemID:             String("test_scan_item_id"),
		ScanItemType:           String("test_scan_item_type"),
		Type:                   String("test_type"),
		UpdatedBefore:          String("test_updated_before"),
		UpdatedAfter:           String("test_updated_after"),
		CreatedBefore:          String("test_created_before"),
		CreatedAfter:           String("test_created_after"),
		EffectiveSeverityLevel: StringList{"test_effective_severity_level_1", "test_effective_severity_level_2"},
		Status:                 StringList{"test_status_1", "test_status_2"},
		Ignored:                Bool(true),
	}
	result, err := client.ListIssues(s.ctx, &req)
	s.NoError(err)
	s.Equal(&ListIssuesRes{
		Data: []any{
			map[string]any{
				"any": "data",
			},
		},
		Links: &Links{
			Next: String("test_next"),
		},
	}, result)
}
