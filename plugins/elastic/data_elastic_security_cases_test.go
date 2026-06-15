// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package elastic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	kbclient_mocks "github.com/blackstork-io/blackstork-cli/mocks/plugins/elastic/kbclient"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugins/elastic/kbclient"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

type ReportsDataSourceTestSuite struct {
	suite.Suite
	schema       *plugin.DataSource
	ctx          context.Context
	cli          *kbclient_mocks.Client
	storedUrl    string
	storedApiKey *string
}

func TestReportsDataSourceTestSuite(t *testing.T) {
	suite.Run(t, new(ReportsDataSourceTestSuite))
}

func (s *ReportsDataSourceTestSuite) SetupSuite() {
	s.schema = makeElasticSecurityCasesDataSource(func(url string, apiKey *string) kbclient.Client {
		s.storedUrl = url
		s.storedApiKey = apiKey
		return s.cli
	})
	s.ctx = context.Background()
}

func (s *ReportsDataSourceTestSuite) SetupTest() {
	s.cli = &kbclient_mocks.Client{}
}

func (s *ReportsDataSourceTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *ReportsDataSourceTestSuite) TestSchema() {
	s.Require().NotNil(s.schema)
	s.NotNil(s.schema.Config)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.DataFunc)
}

func (s *ReportsDataSourceTestSuite) TestAuth() {
	s.cli.On("ListSecurityCases", mock.Anything, &kbclient.ListSecurityCasesReq{
		Page:    1,
		PerPage: 10,
		Search:  kbclient.String("search"),
	}).Return(&kbclient.ListSecurityCasesRes{
		Page:    1,
		PerPage: 10,
		Total:   1,
		Cases: []any{
			map[string]any{
				"id": "1",
			},
		},
	}, nil)
	res, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"cfg"}, map[string]cty.Value{
			"kibana_endpoint_url": cty.StringVal("test_kibana_endpoint_url"),
			"api_key_str":         cty.StringVal("test_api_key_str"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"search": cty.StringVal("search"),
		}),
	})
	s.Equal("test_kibana_endpoint_url", s.storedUrl)
	s.Equal(kbclient.String("test_api_key_str"), s.storedApiKey)
	s.Len(diags, 0)
	s.Equal(plugindata.List{
		plugindata.Map{
			"id": plugindata.String("1"),
		},
	}, res)
}

func (s *ReportsDataSourceTestSuite) TestFull() {
	s.cli.On("ListSecurityCases", mock.Anything, &kbclient.ListSecurityCasesReq{
		SpaceID:               nil,
		Assignees:             []string{"test_assignee_1", "test_assignee_2"},
		DefaultSearchOperator: nil,
		From:                  nil,
		Owner:                 []string{"test_owner_1", "test_owner_2"},
		Page:                  1,
		PerPage:               3,
		Reporters:             []string{"test_reporter_1", "test_reporter_2"},
		Search:                kbclient.String("test_search"),
		SearchFields:          []string{"test_search_field_1", "test_search_field_2"},
		Severity:              kbclient.String("test_severity"),
		SortField:             kbclient.String("test_sort_field"),
		SortOrder:             kbclient.String("test_sort_order"),
		Status:                kbclient.String("test_status"),
		Tags:                  []string{"test_tag_1", "test_tag_2"},
		To:                    kbclient.String("test_to"),
	}).Return(&kbclient.ListSecurityCasesRes{
		Page:    1,
		Total:   2,
		PerPage: 3,
		Cases: []any{
			map[string]any{
				"id": "1",
			},
		},
	}, nil)
	res, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"cfg"}, map[string]cty.Value{
			"kibana_endpoint_url": cty.StringVal("test_kibana_endpoint_url"),
			"api_key_str":         cty.StringVal("test_api_key_str"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"assignees":     cty.ListVal([]cty.Value{cty.StringVal("test_assignee_1"), cty.StringVal("test_assignee_2")}),
			"owner":         cty.ListVal([]cty.Value{cty.StringVal("test_owner_1"), cty.StringVal("test_owner_2")}),
			"reporters":     cty.ListVal([]cty.Value{cty.StringVal("test_reporter_1"), cty.StringVal("test_reporter_2")}),
			"search":        cty.StringVal("test_search"),
			"search_fields": cty.ListVal([]cty.Value{cty.StringVal("test_search_field_1"), cty.StringVal("test_search_field_2")}),
			"severity":      cty.StringVal("test_severity"),
			"sort_field":    cty.StringVal("test_sort_field"),
			"sort_order":    cty.StringVal("test_sort_order"),
			"status":        cty.StringVal("test_status"),
			"tags":          cty.ListVal([]cty.Value{cty.StringVal("test_tag_1"), cty.StringVal("test_tag_2")}),
			"to":            cty.StringVal("test_to"),
			"size":          cty.NumberIntVal(3),
		}),
	})
	s.Len(diags, 0)
	s.Equal(plugindata.List{
		plugindata.Map{
			"id": plugindata.String("1"),
		},
	}, res)
}
