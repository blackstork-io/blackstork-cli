// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package splunk

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	client_mocks "github.com/blackstork-io/blackstork-cli/mocks/plugins/splunk/client"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugins/splunk/client"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

type SearchDataSourceTestSuite struct {
	suite.Suite
	schema           *plugin.DataSource
	ctx              context.Context
	cli              *client_mocks.Client
	storedHost       string
	storedToken      string
	storedDeployment string
}

func TestSearchDataSourceTestSuite(t *testing.T) {
	suite.Run(t, new(SearchDataSourceTestSuite))
}

func (s *SearchDataSourceTestSuite) SetupSuite() {
	s.schema = makeSplunkSearchDataSchema(func(token, host, deployment string) client.Client {
		s.storedHost = host
		s.storedToken = token
		s.storedDeployment = deployment
		return s.cli
	})
	s.ctx = context.Background()
}

func (s *SearchDataSourceTestSuite) SetupTest() {
	s.cli = &client_mocks.Client{}
}

func (s *SearchDataSourceTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *SearchDataSourceTestSuite) TestSchema() {
	s.Require().NotNil(s.schema)
	s.NotNil(s.schema.Config)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.DataFunc)
}

func (s *SearchDataSourceTestSuite) TestSearch() {
	createJobRes := new(client.CreateSearchJobRes)
	getJobReq := &client.GetSearchJobByIDReq{}
	getJobResultsReq := &client.GetSearchJobResultsReq{
		OutputMode: "json",
	}
	s.cli.On("CreateSearchJob", mock.Anything, mock.MatchedBy(func(req *client.CreateSearchJobReq) bool {
		createJobRes.Sid = req.ID
		getJobReq.ID = req.ID
		getJobResultsReq.ID = req.ID
		s.True(strings.HasPrefix(req.ID, "fabric_"), "ID should be prefixed with 'fabric_'")
		s.Equal("test_query", req.Search)
		s.Equal("blocking", req.ExecMode)
		s.Equal(client.Int(1), req.StatusBuckets)
		s.Equal(client.Int(2), req.MaxCount)
		s.Equal([]string{"test_rf_1", "test_rf_2"}, req.RF)
		s.Equal(client.String("test_earliest_time"), req.EarliestTime)
		s.Equal(client.String("test_latest_time"), req.LatestTime)
		return true
	})).
		Return(createJobRes, nil)

	s.cli.On("GetSearchJobByID", mock.Anything, getJobReq).
		Return(&client.GetSearchJobByIDRes{
			DispatchState: client.DispatchStateDone,
		}, nil)

	s.cli.On("GetSearchJobResults", mock.Anything, getJobResultsReq).
		Return(&client.GetSearchJobResultsRes{
			Results: []any{
				map[string]any{"key": "value"},
			},
		}, nil)

	data, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"auth_token":      cty.StringVal("test_token"),
			"host":            cty.StringVal("test_host"),
			"deployment_name": cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"search_query":   cty.StringVal("test_query"),
			"status_buckets": cty.NumberIntVal(1),
			"max_count":      cty.NumberIntVal(2),
			"rf":             cty.ListVal([]cty.Value{cty.StringVal("test_rf_1"), cty.StringVal("test_rf_2")}),
			"earliest_time":  cty.StringVal("test_earliest_time"),
			"latest_time":    cty.StringVal("test_latest_time"),
		}),
	})
	s.Require().Len(diags, 0)
	s.Equal(plugindata.List{
		plugindata.Map{
			"key": plugindata.String("value"),
		},
	}, data)

	s.Equal("test_token", s.storedToken)
	s.Equal("test_host", s.storedHost)
	s.Empty(s.storedDeployment)
}

func (s *SearchDataSourceTestSuite) TestSearchError() {
	resErr := fmt.Errorf("test error")
	s.cli.On("CreateSearchJob", mock.Anything, mock.Anything).
		Return(nil, resErr)

	_, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"auth_token":      cty.StringVal("test_token"),
			"host":            cty.StringVal("test_host"),
			"deployment_name": cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"search_query":   cty.StringVal("test_query"),
			"status_buckets": cty.NumberIntVal(1),
			"max_count":      cty.NumberIntVal(2),
			"rf":             cty.ListVal([]cty.Value{cty.StringVal("test_rf_1"), cty.StringVal("test_rf_2")}),
			"earliest_time":  cty.StringVal("test_earliest_time"),
			"latest_time":    cty.StringVal("test_latest_time"),
		}),
	})
	s.Require().Len(diags, 1)
	s.Equal("Failed to search", diags[0].Summary)
}

func (s *SearchDataSourceTestSuite) TestSearchJobError() {
	createJobRes := new(client.CreateSearchJobRes)
	resErr := fmt.Errorf("test error")
	s.cli.On("CreateSearchJob", mock.Anything, mock.MatchedBy(func(req *client.CreateSearchJobReq) bool {
		createJobRes.Sid = req.ID
		return true
	})).
		Return(createJobRes, nil)

	s.cli.On("GetSearchJobByID", mock.Anything, mock.Anything).
		Return(nil, resErr)

	_, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"auth_token":      cty.StringVal("test_token"),
			"host":            cty.StringVal("test_host"),
			"deployment_name": cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"search_query":   cty.StringVal("test_query"),
			"status_buckets": cty.NumberIntVal(1),
			"max_count":      cty.NumberIntVal(2),
			"rf":             cty.ListVal([]cty.Value{cty.StringVal("test_rf_1"), cty.StringVal("test_rf_2")}),
			"earliest_time":  cty.StringVal("test_earliest_time"),
			"latest_time":    cty.StringVal("test_latest_time"),
		}),
	})
	s.Require().Len(diags, 1)
	s.Equal("Failed to search", diags[0].Summary)
}

func (s *SearchDataSourceTestSuite) TestSearchJobResultsError() {
	createJobRes := new(client.CreateSearchJobRes)
	getJobReq := &client.GetSearchJobByIDReq{}
	getJobResultsReq := &client.GetSearchJobResultsReq{
		OutputMode: "json",
	}
	s.cli.On("CreateSearchJob", mock.Anything, mock.MatchedBy(func(req *client.CreateSearchJobReq) bool {
		createJobRes.Sid = req.ID
		getJobReq.ID = req.ID
		getJobResultsReq.ID = req.ID
		return true
	})).
		Return(createJobRes, nil)

	s.cli.On("GetSearchJobByID", mock.Anything, getJobReq).
		Return(&client.GetSearchJobByIDRes{
			DispatchState: client.DispatchStateDone,
		}, nil)

	resErr := fmt.Errorf("test error")
	s.cli.On("GetSearchJobResults", mock.Anything, getJobResultsReq).
		Return(nil, resErr)

	_, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"auth_token":      cty.StringVal("test_token"),
			"host":            cty.StringVal("test_host"),
			"deployment_name": cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"search_query":   cty.StringVal("test_query"),
			"status_buckets": cty.NumberIntVal(1),
			"max_count":      cty.NumberIntVal(2),
			"rf":             cty.ListVal([]cty.Value{cty.StringVal("test_rf_1"), cty.StringVal("test_rf_2")}),
			"earliest_time":  cty.StringVal("test_earliest_time"),
			"latest_time":    cty.StringVal("test_latest_time"),
		}),
	})
	s.Require().Len(diags, 1)
	s.Equal("Failed to search", diags[0].Summary)
}

func (s *SearchDataSourceTestSuite) TestSearchEmptyQuery() {
	_, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"auth_token":      cty.StringVal("test_token"),
			"host":            cty.StringVal("test_host"),
			"deployment_name": cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"search_query":   cty.StringVal(""),
			"status_buckets": cty.NumberIntVal(1),
			"max_count":      cty.NumberIntVal(2),
			"rf":             cty.ListVal([]cty.Value{cty.StringVal("test_rf_1"), cty.StringVal("test_rf_2")}),
			"earliest_time":  cty.StringVal("test_earliest_time"),
			"latest_time":    cty.StringVal("test_latest_time"),
		}),
	})
	s.Require().Len(diags, 1)
	s.Equal("search_query is required", diags[0].Detail)
}
