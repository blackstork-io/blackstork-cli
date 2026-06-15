// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package iris

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	client_mocks "github.com/blackstork-io/blackstork-cli/mocks/plugins/iris/client"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugin/plugintest"
	"github.com/blackstork-io/blackstork-cli/plugins/iris/client"
)

type AlertsDataSourceTestSuite struct {
	suite.Suite

	plugin         *plugin.Schema
	ctx            context.Context
	cli            *client_mocks.Client
	storedApiURL   string
	storedApiKey   string
	storedInsecure bool
}

func TestAlertsDataSourceTestSuite(t *testing.T) {
	suite.Run(t, new(AlertsDataSourceTestSuite))
}

func (s *AlertsDataSourceTestSuite) SetupSuite() {
	s.plugin = Plugin("v0.0.0", func(apiURL, apiKey string, insecure bool) client.Client {
		s.storedApiKey = apiKey
		s.storedApiURL = apiURL
		s.storedInsecure = insecure
		return s.cli
	})
	s.ctx = context.Background()
}

func (s *AlertsDataSourceTestSuite) SetupTest() {
	s.cli = &client_mocks.Client{}
}

func (s *AlertsDataSourceTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *AlertsDataSourceTestSuite) TestSchema() {
	s.Require().NotNil(s.plugin.DataSources["iris_alerts"])
	s.NotNil(s.plugin.DataSources["iris_alerts"].Config)
	s.NotNil(s.plugin.DataSources["iris_alerts"].Args)
	s.NotNil(s.plugin.DataSources["iris_alerts"].DataFunc)
}

func (s *AlertsDataSourceTestSuite) TestLimit() {
	s.cli.On("ListAlerts", mock.Anything, &client.ListAlertsReq{
		Page: 1,
		Sort: client.String("desc"),
	}).Return(&client.ListAlertsRes{
		Status: "success",
		Data: &client.AlertsData{
			CurrentPage: 1,
			LastPage:    1,
			Total:       1,
			Alerts: []any{
				map[string]any{
					"id": "1",
				},
			},
		},
	}, nil)
	res, diags := s.plugin.RetrieveData(s.ctx, "iris_alerts", &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["iris_alerts"].Config).
			SetAttr("api_url", cty.StringVal("test-url")).
			SetAttr("api_key", cty.StringVal("test-key")).
			SetAttr("insecure", cty.BoolVal(true)).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["iris_alerts"].Args).
			SetAttr("size", cty.NumberIntVal(10)).
			Decode(),
	})
	s.Equal("test-url", s.storedApiURL)
	s.Equal("test-key", s.storedApiKey)
	s.Equal(true, s.storedInsecure)
	s.Len(diags, 0)
	s.Equal(plugindata.List{
		plugindata.Map{
			"id": plugindata.String("1"),
		},
	}, res)
}

func (s *AlertsDataSourceTestSuite) TestFull() {
	s.cli.On("ListAlerts", mock.Anything, &client.ListAlertsReq{
		Page:                  1,
		AlertIDs:              client.IntList{1, 2},
		AlertCustomerID:       client.Int(1),
		AlertOwnerID:          client.Int(2),
		AlertSeverityID:       client.Int(5),
		CaseID:                client.Int(3),
		AlertTags:             client.StringList{"test-tag-1", "test-tag-2"},
		AlertSource:           client.String("test-source"),
		AlertStatusID:         client.Int(4),
		AlertClassificationID: client.Int(5),
		AlertStartDate:        client.String("test-alert-start-date"),
		AlertEndDate:          client.String("test-alert-end-date"),
		Sort:                  client.String("asc"),
	}).Return(&client.ListAlertsRes{
		Status: "success",
		Data: &client.AlertsData{
			CurrentPage: 1,
			LastPage:    2,
			Total:       3,
			NextPage:    client.Int(2),
			Alerts: []any{
				map[string]any{
					"id": "1",
				},
			},
		},
	}, nil)
	s.cli.On("ListAlerts", mock.Anything, &client.ListAlertsReq{
		Page:                  2,
		AlertIDs:              client.IntList{1, 2},
		AlertCustomerID:       client.Int(1),
		AlertOwnerID:          client.Int(2),
		AlertSeverityID:       client.Int(5),
		CaseID:                client.Int(3),
		AlertTags:             client.StringList{"test-tag-1", "test-tag-2"},
		AlertSource:           client.String("test-source"),
		AlertStatusID:         client.Int(4),
		AlertClassificationID: client.Int(5),
		AlertStartDate:        client.String("test-alert-start-date"),
		AlertEndDate:          client.String("test-alert-end-date"),
		Sort:                  client.String("asc"),
	}).Return(&client.ListAlertsRes{
		Status: "success",
		Data: &client.AlertsData{
			CurrentPage: 2,
			LastPage:    2,
			Total:       3,
			Alerts: []any{
				map[string]any{
					"id": "2",
				},
				map[string]any{
					"id": "3",
				},
			},
		},
	}, nil)
	res, diags := s.plugin.RetrieveData(s.ctx, "iris_alerts", &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["iris_alerts"].Config).
			SetAttr("api_url", cty.StringVal("test-url")).
			SetAttr("api_key", cty.StringVal("test-key")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["iris_alerts"].Args).
			SetAttr("alert_ids", cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
			})).
			SetAttr("tags", cty.ListVal([]cty.Value{
				cty.StringVal("test-tag-1"),
				cty.StringVal("test-tag-2"),
			})).
			SetAttr("case_id", cty.NumberIntVal(3)).
			SetAttr("customer_id", cty.NumberIntVal(1)).
			SetAttr("owner_id", cty.NumberIntVal(2)).
			SetAttr("severity_id", cty.NumberIntVal(5)).
			SetAttr("status_id", cty.NumberIntVal(4)).
			SetAttr("classification_id", cty.NumberIntVal(5)).
			SetAttr("alert_source", cty.StringVal("test-source")).
			SetAttr("alert_start_date", cty.StringVal("test-alert-start-date")).
			SetAttr("alert_end_date", cty.StringVal("test-alert-end-date")).
			SetAttr("sort", cty.StringVal("asc")).
			SetAttr("size", cty.NumberIntVal(2)).
			Decode(),
	})
	s.Equal("test-url", s.storedApiURL)
	s.Equal("test-key", s.storedApiKey)
	s.Equal(false, s.storedInsecure)
	s.Len(diags, 0)
	s.Equal(plugindata.List{
		plugindata.Map{
			"id": plugindata.String("1"),
		},
		plugindata.Map{
			"id": plugindata.String("2"),
		},
	}, res)
}
