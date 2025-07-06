package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func (s *ClientTestSuite) mock(fn http.HandlerFunc, usr, tkn string) (Client, *httptest.Server) {
	srv := httptest.NewServer(fn)
	cli := &client{
		url: srv.URL,
		usr: usr,
		tkn: tkn,
	}
	return cli, srv
}

func (s *ClientTestSuite) TestAuth() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		usr, tkn, ok := r.BasicAuth()
		s.True(ok)
		s.Equal("test_user", usr)
		s.Equal("test_token", tkn)
	}, "test_user", "test_token")
	defer srv.Close()
	client.GetAllReports(s.ctx, &GetAllReportsReq{})
}

func (s *ClientTestSuite) queryList(q url.Values, key string) []string {
	list, ok := q[key]
	s.Require().True(ok)
	return list
}

func (s *ClientTestSuite) TestGetAllReports() {
	ts := time.Unix(123, 0).UTC()
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/v1/reports", r.URL.Path)
		s.Equal(http.MethodGet, r.Method)
		usr, tkn, ok := r.BasicAuth()
		s.True(ok)
		s.Equal("test_user", usr)
		s.Equal("test_token", tkn)
		s.Equal("10", r.URL.Query().Get("page[size]"))
		s.Equal("1", r.URL.Query().Get("page[number]"))
		s.Equal("test_h1b", r.URL.Query().Get("filter[program][]"))
		s.Equal("72049", r.URL.Query().Get("filter[inbox_ids][]"))
		s.Equal("created_at", r.URL.Query().Get("sort"))
		s.Equal("test_reporter", r.URL.Query().Get("filter[reporter][]"))
		s.Equal("test_assignee", r.URL.Query().Get("filter[assignee][]"))
		s.Equal("test_state", r.URL.Query().Get("filter[state][]"))
		s.Equal([]string{"1", "2", "3"}, s.queryList(r.URL.Query(), "filter[id][]"))
		s.Equal([]string{"1", "2", "3"}, s.queryList(r.URL.Query(), "filter[weakness_id][]"))
		s.Equal("test_severity", r.URL.Query().Get("filter[severity][]"))
		s.Equal("true", r.URL.Query().Get("filter[hacker_published]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[created_at__gt]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[created_at__lt]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[submitted_at__gt]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[submitted_at__lt]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[triaged_at__gt]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[triaged_at__lt]"))
		s.Equal("true", r.URL.Query().Get("filter[triaged_at__null]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[closed_at__gt]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[closed_at__lt]"))
		s.Equal("true", r.URL.Query().Get("filter[closed_at__null]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[disclosed_at__gt]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[disclosed_at__lt]"))
		s.Equal("true", r.URL.Query().Get("filter[disclosed_at__null]"))
		s.Equal("true", r.URL.Query().Get("filter[reporter_agreed_on_going_public]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[bounty_awarded_at__gt]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[bounty_awarded_at__lt]"))
		s.Equal("true", r.URL.Query().Get("filter[bounty_awarded_at__null]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[swag_awarded_at__gt]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[swag_awarded_at__lt]"))
		s.Equal("true", r.URL.Query().Get("filter[swag_awarded_at__null]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[last_report_activity_at__gt]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[last_report_activity_at__lt]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[first_program_activity_at__gt]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[first_program_activity_at__lt]"))
		s.Equal("true", r.URL.Query().Get("filter[first_program_activity_at__null]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[last_program_activity_at__gt]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[last_program_activity_at__lt]"))
		s.Equal("true", r.URL.Query().Get("filter[last_program_activity_at__null]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[last_activity_at__gt]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[last_activity_at__lt]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[last_public_activity_at__gt]"))
		s.Equal("1970-01-01T00:02:03Z", r.URL.Query().Get("filter[last_public_activity_at__lt]"))
		s.Equal("test_keyword", r.URL.Query().Get("filter[keyword]"))
		s.Equal("map[test_key:test_value]", r.URL.Query().Get("filter[custom_fields][]"))
		w.Write([]byte(`{
			"data": [
				{
					"any": "data"
				}
			]
		}`))
	}, "test_user", "test_token")
	defer srv.Close()
	req := GetAllReportsReq{
		PageSize:                          Int(10),
		PageNumber:                        Int(1),
		FilterProgram:                     []string{"test_h1b"},
		FilterInboxIDs:                    []int{72049},
		Sort:                              String("created_at"),
		FilterReporter:                    []string{"test_reporter"},
		FilterAssignee:                    []string{"test_assignee"},
		FilterState:                       []string{"test_state"},
		FilterID:                          []int{1, 2, 3},
		FilterWeaknessID:                  []int{1, 2, 3},
		FilterSeverity:                    []string{"test_severity"},
		FilterHackerPublished:             Bool(true),
		FilterCreatedAtGT:                 &ts,
		FilterCreatedAtLT:                 &ts,
		FilterSubmittedAtGT:               &ts,
		FilterSubmittedAtLT:               &ts,
		FilterTriagedAtGT:                 &ts,
		FilterTriagedAtLT:                 &ts,
		FilterTriagedAtNull:               Bool(true),
		FilterClosedAtGT:                  &ts,
		FilterClosedAtLT:                  &ts,
		FilterClosedAtNull:                Bool(true),
		FilterDisclosedAtGT:               &ts,
		FilterDisclosedAtLT:               &ts,
		FilterDisclosedAtNull:             Bool(true),
		FilterReporterAgreedOnGoingPublic: Bool(true),
		FilterBountyAwardedAtGT:           &ts,
		FilterBountyAwardedAtLT:           &ts,
		FilterBountyAwardedAtNull:         Bool(true),
		FilterSwagAwardedAtGT:             &ts,
		FilterSwagAwardedAtLT:             &ts,
		FilterSwagAwardedAtNull:           Bool(true),
		FilterLastReportActivityAtGT:      &ts,
		FilterLastReportActivityAtLT:      &ts,
		FilterFirstProgramActivityAtGT:    &ts,
		FilterFirstProgramActivityAtLT:    &ts,
		FilterFirstProgramActivityAtNull:  Bool(true),
		FilterLastProgramActivityAtGT:     &ts,
		FilterLastProgramActivityAtLT:     &ts,
		FilterLastProgramActivityAtNull:   Bool(true),
		FilterLastActivityAtGT:            &ts,
		FilterLastActivityAtLT:            &ts,
		FilterLastPublicActivityAtGT:      &ts,
		FilterLastPublicActivityAtLT:      &ts,
		FilterKeyword:                     String("test_keyword"),
		FilterCustomFields: map[string]string{
			"test_key": "test_value",
		},
	}
	result, err := client.GetAllReports(s.ctx, &req)
	s.NoError(err)
	s.Equal(&GetAllReportsRes{
		Data: []any{
			map[string]any{
				"any": "data",
			},
		},
	}, result)
}

func (s *ClientTestSuite) TestGetAllReportsError() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}, "test_user", "test_token")
	defer srv.Close()
	req := GetAllReportsReq{}
	_, err := client.GetAllReports(s.ctx, &req)
	s.Error(err)
}

func (s *ClientTestSuite) TestDefaultClientURL() {
	cli := New("test_user", "test_token")
	s.Equal("https://api.hackerone.com", cli.(*client).url)
}
