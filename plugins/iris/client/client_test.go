package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func (s *ClientTestSuite) mock(fn http.HandlerFunc, apiKey string, insecure bool) (Client, *httptest.Server) {
	srv := httptest.NewServer(fn)
	cli := &client{
		apiURL:   srv.URL,
		apiKey:   apiKey,
		insecure: insecure,
	}
	return cli, srv
}

func (s *ClientTestSuite) TestAuth() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("api_key", strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	}, "api_key", false)
	defer srv.Close()
	client.ListCases(s.ctx, &ListCasesReq{})
}

func (s *ClientTestSuite) TestListCases() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/manage/cases/filter", r.URL.Path)
		s.Equal(http.MethodGet, r.Method)
		// page
		q := r.URL.Query()
		s.Equal("1", q.Get("page"))
		s.Equal("1", q.Get("per_page"))
		s.Equal("1,2", q.Get("case_ids"))
		s.Equal("1", q.Get("case_customer_id"))
		s.Equal("1", q.Get("case_owner_id"))
		s.Equal("1", q.Get("case_severity_id"))
		s.Equal("1", q.Get("case_state_id"))
		s.Equal("test_soc_id", q.Get("case_soc_id"))
		s.Equal("asc", q.Get("sort"))
		s.Equal("test_start_date", q.Get("start_open_date"))
		s.Equal("test_end_date", q.Get("end_open_date"))
		w.Write([]byte(`{
			"status": "success",
			"data": {
				"last_page": 1,
				"current_page": 1,
				"total": 10,
				"next_page": 2,
				"cases": [
					{
						"any": "data"
					}
				]
			}
		}`))
	}, "test_api_key", false)
	defer srv.Close()
	req := ListCasesReq{
		Page:           1,
		PerPage:        Int(1),
		CaseIDs:        IntList{1, 2},
		CaseCustomerID: Int(1),
		CaseOwnerID:    Int(1),
		CaseSeverityID: Int(1),
		CaseStateID:    Int(1),
		CaseSocID:      String("test_soc_id"),
		Sort:           String("asc"),
		StartOpenDate:  String("test_start_date"),
		EndOpenDate:    String("test_end_date"),
	}
	result, err := client.ListCases(s.ctx, &req)
	s.NoError(err)
	s.Equal(&ListCasesRes{
		Status: "success",
		Data: &CasesData{
			CurrentPage: 1,
			LastPage:    1,
			NextPage:    Int(2),
			Total:       10,
			Cases: []any{
				map[string]any{
					"any": "data",
				},
			},
		},
	}, result)
}

func (s *ClientTestSuite) TestListCasesError() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}, "test_api_key", false)
	defer srv.Close()
	req := ListCasesReq{}
	_, err := client.ListCases(s.ctx, &req)
	s.Error(err)
}

func (s *ClientTestSuite) TestListAlerts() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/alerts/filter", r.URL.Path)
		s.Equal(http.MethodGet, r.Method)
		// page
		q := r.URL.Query()
		s.Equal("1", q.Get("page"))
		s.Equal("1", q.Get("per_page"))
		s.Equal("1,2", q.Get("alert_ids"))
		s.Equal("1", q.Get("case_id"))
		s.Equal("1", q.Get("alert_customer_id"))
		s.Equal("1", q.Get("alert_owner_id"))
		s.Equal("1", q.Get("alert_status_id"))
		s.Equal("1", q.Get("alert_classification_id"))
		s.Equal("test_tag_1,test_tag_2", q.Get("alert_tags"))
		s.Equal("1", q.Get("alert_severity_id"))
		s.Equal("test_alert_source", q.Get("alert_source"))
		s.Equal("asc", q.Get("sort"))
		s.Equal("test_start_date", q.Get("alert_start_date"))
		s.Equal("test_end_date", q.Get("alert_end_date"))
		w.Write([]byte(`{
			"status": "success",
			"data": {
				"last_page": 1,
				"current_page": 1,
				"total": 10,
				"next_page": 2,
				"alerts": [
					{
						"any": "data"
					}
				]
			}
		}`))
	}, "test_api_key", false)
	defer srv.Close()
	req := ListAlertsReq{
		Page:                  1,
		PerPage:               Int(1),
		AlertIDs:              IntList{1, 2},
		AlertTags:             StringList{"test_tag_1", "test_tag_2"},
		AlertCustomerID:       Int(1),
		AlertOwnerID:          Int(1),
		CaseID:                Int(1),
		AlertClassificationID: Int(1),
		AlertSeverityID:       Int(1),
		AlertSource:           String("test_alert_source"),
		AlertStatusID:         Int(1),
		Sort:                  String("asc"),
		AlertStartDate:        String("test_start_date"),
		AlertEndDate:          String("test_end_date"),
	}
	result, err := client.ListAlerts(s.ctx, &req)
	s.NoError(err)
	s.Equal(&ListAlertsRes{
		Status: "success",
		Data: &AlertsData{
			CurrentPage: 1,
			LastPage:    1,
			NextPage:    Int(2),
			Total:       10,
			Alerts: []any{
				map[string]any{
					"any": "data",
				},
			},
		},
	}, result)
}

func (s *ClientTestSuite) TestListAlertsError() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}, "test_api_key", false)
	defer srv.Close()
	req := ListAlertsReq{}
	_, err := client.ListAlerts(s.ctx, &req)
	s.Error(err)
}
