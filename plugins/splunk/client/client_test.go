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

func (s *ClientTestSuite) mock(fn http.HandlerFunc, tkn string) (Client, *httptest.Server) {
	srv := httptest.NewServer(fn)
	cli := &client{
		url:   srv.URL,
		token: tkn,
	}
	return cli, srv
}

func (s *ClientTestSuite) TestAuth() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("Bearer test_token", r.Header.Get("Authorization"))
	}, "test_token")
	defer srv.Close()
	client.GetSearchJobByID(s.ctx, &GetSearchJobByIDReq{})
}

func (s *ClientTestSuite) TestCreateSearchJob() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/services/search/jobs", r.URL.Path)
		s.Equal("Bearer test_token", r.Header.Get("Authorization"))
		err := r.ParseForm()
		s.Require().NoError(err)
		s.Equal("test_id", r.FormValue("id"))
		s.Equal("test_exec_mode", r.FormValue("exec_mode"))
		s.Equal("test_search", r.FormValue("search"))
		s.Equal("1", r.FormValue("status_buckets"))
		s.Equal("2", r.FormValue("max_count"))
		s.Equal("test_rf", r.Form["rf"][0])
		s.Equal("test_rf", r.Form["rf"][1])
		s.Equal("test_earliest_time", r.FormValue("earliest_time"))
		s.Equal("test_latest_time", r.FormValue("latest_time"))
		w.Write([]byte(`{"sid":"test_sid"}`))
	}, "test_token")
	defer srv.Close()
	res, err := client.CreateSearchJob(s.ctx, &CreateSearchJobReq{
		ID:            "test_id",
		ExecMode:      "test_exec_mode",
		Search:        "test_search",
		StatusBuckets: Int(1),
		MaxCount:      Int(2),
		RF:            []string{"test_rf", "test_rf"},
		EarliestTime:  String("test_earliest_time"),
		LatestTime:    String("test_latest_time"),
	})
	s.Require().NoError(err)
	s.Equal("test_sid", res.Sid)
}

func (s *ClientTestSuite) TestCreateSearchJobError() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}, "test_token")
	defer srv.Close()
	_, err := client.CreateSearchJob(s.ctx, &CreateSearchJobReq{})
	s.Require().Error(err)
}

func (s *ClientTestSuite) TestGetSearchJobByID() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/services/search/jobs/test_id", r.URL.Path)
		s.Equal("Bearer test_token", r.Header.Get("Authorization"))
		w.Write([]byte(`{"dispatchState":"QUEUED"}`))
	}, "test_token")
	defer srv.Close()
	res, err := client.GetSearchJobByID(s.ctx, &GetSearchJobByIDReq{ID: "test_id"})
	s.Require().NoError(err)
	s.Equal(DispatchStateQueued, res.DispatchState)
}

func (s *ClientTestSuite) TestGetSearchJobByIDError() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}, "test_token")
	defer srv.Close()
	_, err := client.GetSearchJobByID(s.ctx, &GetSearchJobByIDReq{ID: "test_id"})
	s.Require().Error(err)
}

func (s *ClientTestSuite) TestDispatchStateWait() {
	s.True(DispatchStateQueued.Wait())
	s.True(DispatchStateParsing.Wait())
	s.True(DispatchStateRunning.Wait())
	s.True(DispatchStateFinalizing.Wait())
	s.False(DispatchStateDone.Wait())
	s.False(DispatchStatePause.Wait())
	s.False(DispatchStateInternalCancel.Wait())
	s.False(DispatchStateUserCancel.Wait())
	s.False(DispatchStateBadInputCancel.Wait())
	s.False(DispatchStateQuit.Wait())
	s.False(DispatchStateFailed.Wait())
}

func (s *ClientTestSuite) TestDispatchStateDone() {
	s.True(DispatchStateDone.Done())
	s.False(DispatchStateQueued.Done())
	s.False(DispatchStateParsing.Done())
	s.False(DispatchStateRunning.Done())
	s.False(DispatchStateFinalizing.Done())
	s.False(DispatchStatePause.Done())
	s.False(DispatchStateInternalCancel.Done())
	s.False(DispatchStateUserCancel.Done())
	s.False(DispatchStateBadInputCancel.Done())
	s.False(DispatchStateQuit.Done())
	s.False(DispatchStateFailed.Done())
}

func (s *ClientTestSuite) TestDispatchStateFailed() {
	s.False(DispatchStateQueued.Failed())
	s.False(DispatchStateParsing.Failed())
	s.False(DispatchStateRunning.Failed())
	s.False(DispatchStateFinalizing.Failed())
	s.False(DispatchStateDone.Failed())
	s.True(DispatchStatePause.Failed())
	s.True(DispatchStateFailed.Failed())
	s.True(DispatchStateInternalCancel.Failed())
	s.True(DispatchStateUserCancel.Failed())
	s.True(DispatchStateBadInputCancel.Failed())
	s.True(DispatchStateQuit.Failed())
}

func (s *ClientTestSuite) TestGetSearchJobResults() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/services/search/v2/jobs/test_id/results", r.URL.Path)
		s.Equal("output_mode=json", r.URL.RawQuery)
		s.Equal("Bearer test_token", r.Header.Get("Authorization"))
		w.Write([]byte(`{"results":[{"test_key":"test_value"}]}`))
	}, "test_token")
	defer srv.Close()
	res, err := client.GetSearchJobResults(s.ctx, &GetSearchJobResultsReq{
		ID:         "test_id",
		OutputMode: "json",
	})
	s.Require().NoError(err)
	s.Equal([]any{
		map[string]any{"test_key": "test_value"},
	}, res.Results)
}

func (s *ClientTestSuite) TestGetSearchJobResultsError() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}, "test_token")
	defer srv.Close()
	_, err := client.GetSearchJobResults(s.ctx, &GetSearchJobResultsReq{ID: "test_id"})
	s.Require().Error(err)
}
