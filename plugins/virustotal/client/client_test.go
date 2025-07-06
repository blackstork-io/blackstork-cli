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

func (s *ClientTestSuite) mock(fn http.HandlerFunc, tkn string) (*client, *httptest.Server) {
	srv := httptest.NewServer(fn)
	cli := &client{
		url: srv.URL,
		key: tkn,
	}
	return cli, srv
}

func (s *ClientTestSuite) TestAuth() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("test_token", r.Header.Get("x-apikey"))
	}, "test_token")
	defer srv.Close()
	client.GetUserAPIUsage(s.ctx, &GetUserAPIUsageReq{User: "test_user"})
}

func (s *ClientTestSuite) TestGetUserAPIUsageWithQuery() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("test_token", r.Header.Get("x-apikey"))
		s.Equal("GET", r.Method)
		s.Equal("/users/test_user/api_usage", r.URL.Path)
		s.Equal("20240101", r.URL.Query().Get("start_date"))
		s.Equal("20240103", r.URL.Query().Get("end_date"))
		w.Write([]byte(`{"data": {
			"daily": {
				"2024-01-01": {},
				"2024-01-02": {},
				"2024-01-03": {}
			}}}`))
	}, "test_token")
	defer srv.Close()
	start, err := time.Parse("20060102", "20240101")
	s.Require().NoError(err)
	end, err := time.Parse("20060102", "20240103")
	s.Require().NoError(err)
	res, err := client.GetUserAPIUsage(s.ctx, &GetUserAPIUsageReq{
		User:      "test_user",
		StartDate: &Date{start},
		EndDate:   &Date{end},
	})
	s.Require().NoError(err)
	s.Equal(map[string]any{
		"daily": map[string]any{
			"2024-01-01": map[string]any{},
			"2024-01-02": map[string]any{},
			"2024-01-03": map[string]any{},
		},
	}, res.Data)
}

func (s *ClientTestSuite) TestGetUserAPIUsage() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("test_token", r.Header.Get("x-apikey"))
		s.Equal("GET", r.Method)
		s.Equal("/users/test_user/api_usage", r.URL.Path)
		w.Write([]byte(`{"data": {
			"daily": {
				"2024-01-01": {},
				"2024-01-02": {},
				"2024-01-03": {}
			}}}`))
	}, "test_token")
	defer srv.Close()
	res, err := client.GetUserAPIUsage(s.ctx, &GetUserAPIUsageReq{
		User: "test_user",
	})
	s.Require().NoError(err)
	s.Equal(map[string]any{
		"daily": map[string]any{
			"2024-01-01": map[string]any{},
			"2024-01-02": map[string]any{},
			"2024-01-03": map[string]any{},
		},
	}, res.Data)
}

func (s *ClientTestSuite) TestGetGroupAPIUsageWithQuery() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("test_token", r.Header.Get("x-apikey"))
		s.Equal("GET", r.Method)
		s.Equal("/groups/test_group/api_usage", r.URL.Path)
		s.Equal("20240101", r.URL.Query().Get("start_date"))
		s.Equal("20240103", r.URL.Query().Get("end_date"))
		w.Write([]byte(`{"data": {
			"daily": {
				"2024-01-01": {},
				"2024-01-02": {},
				"2024-01-03": {}
			}}}`))
	}, "test_token")
	defer srv.Close()
	start, err := time.Parse("20060102", "20240101")
	s.Require().NoError(err)
	end, err := time.Parse("20060102", "20240103")
	s.Require().NoError(err)
	res, err := client.GetGroupAPIUsage(s.ctx, &GetGroupAPIUsageReq{
		Group:     "test_group",
		StartDate: &Date{start},
		EndDate:   &Date{end},
	})
	s.Require().NoError(err)
	s.Equal(map[string]any{
		"daily": map[string]any{
			"2024-01-01": map[string]any{},
			"2024-01-02": map[string]any{},
			"2024-01-03": map[string]any{},
		},
	}, res.Data)
}

func (s *ClientTestSuite) TestGetGroupAPIUsage() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("test_token", r.Header.Get("x-apikey"))
		s.Equal("GET", r.Method)
		s.Equal("/groups/test_group/api_usage", r.URL.Path)
		w.Write([]byte(`{"data": {
			"daily": {
				"2024-01-01": {},
				"2024-01-02": {},
				"2024-01-03": {}
			}}}`))
	}, "test_token")
	defer srv.Close()
	res, err := client.GetGroupAPIUsage(s.ctx, &GetGroupAPIUsageReq{
		Group: "test_group",
	})
	s.Require().NoError(err)
	s.Equal(map[string]any{
		"daily": map[string]any{
			"2024-01-01": map[string]any{},
			"2024-01-02": map[string]any{},
			"2024-01-03": map[string]any{},
		},
	}, res.Data)
}

func (s *ClientTestSuite) TestGetUserAPIUsageError() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("test_token", r.Header.Get("x-apikey"))
		s.Equal("GET", r.Method)
		s.Equal("/users/test_user/api_usage", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {
			"code": "test_code",
			"message": "test_message"
		}}`))
	}, "test_token")
	defer srv.Close()
	_, err := client.GetUserAPIUsage(s.ctx, &GetUserAPIUsageReq{
		User: "test_user",
	})
	s.Require().Error(err)
	s.Equal("test_code: test_message", err.Error())
}

func (s *ClientTestSuite) TestGetGroupAPIUsageError() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("test_token", r.Header.Get("x-apikey"))
		s.Equal("GET", r.Method)
		s.Equal("/groups/test_group/api_usage", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {
			"code": "test_code",
			"message": "test_message"
		}}`))
	}, "test_token")
	defer srv.Close()
	_, err := client.GetGroupAPIUsage(s.ctx, &GetGroupAPIUsageReq{
		Group: "test_group",
	})
	s.Require().Error(err)
	s.Equal("test_code: test_message", err.Error())
}
