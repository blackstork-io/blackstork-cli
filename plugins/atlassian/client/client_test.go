package client

import (
	"context"
	"io"
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

func (s *ClientTestSuite) mock(fn http.HandlerFunc, accountEmail, apiToken string) (Client, *httptest.Server) {
	srv := httptest.NewServer(fn)
	cli := New(srv.URL, accountEmail, apiToken)
	return cli, srv
}

func (s *ClientTestSuite) TestSearchIssuesC() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/rest/api/3/search/jql", r.URL.Path)
		s.Equal(http.MethodPost, r.Method)
		s.Equal("application/json", r.Header.Get("Content-Type"))
		s.Equal("application/json", r.Header.Get("Accept"))

		user, pass, ok := r.BasicAuth()
		s.True(ok)
		s.Equal("test-email", user)
		s.Equal("test-token", pass)

		body, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		defer r.Body.Close()
		s.JSONEq(`{
			"expand": "names",
			"fields": [
				"*all"
			],
			"jql": "project = TEST",
			"maxResults": 15,
			"properties": ["test_property_1"],
			"nextPageToken": "test_page_token_1"
		}`, string(body))
		w.Write([]byte(`{
			"nextPageToken": "test_page_token_2",
			"issues": [
				{
					"any": "data"
				}
			]
		}`))
	}, "test-email", "test-token")
	defer srv.Close()

	req := SearchIssuesReq{
		Expand:        String("names"),
		Fields:        []string{"*all"},
		JQL:           String("project = TEST"),
		MaxResults:    Int(15),
		Properties:    []string{"test_property_1"},
		NextPageToken: String("test_page_token_1"),
	}

	result, err := client.SearchIssues(s.ctx, &req)
	s.NoError(err)
	s.Equal(&SearchIssuesRes{
		NextPageToken: String("test_page_token_2"),
		Issues: []any{
			map[string]any{
				"any": "data",
			},
		},
	}, result)
}

func (s *ClientTestSuite) TestSearchIssuesError() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{
			"errorMessages" : [
			   "Test Error"
			]
 		}`))
	}, "", "")
	defer srv.Close()
	_, err := client.SearchIssues(s.ctx, &SearchIssuesReq{})
	s.Equal(&Error{
		ErrorMessages: []string{"Test Error"},
	}, err)
}
