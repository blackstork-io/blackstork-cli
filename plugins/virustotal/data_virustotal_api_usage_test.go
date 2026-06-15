// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package virustotal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	client_mocks "github.com/blackstork-io/blackstork-cli/mocks/plugins/virustotal/client"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugin/plugintest"
	"github.com/blackstork-io/blackstork-cli/plugins/virustotal/client"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

type APIUsageTestSuite struct {
	suite.Suite
	schema    *plugin.DataSource
	ctx       context.Context
	cli       *client_mocks.Client
	storedTkn string
}

func TestAPIUsageTestSuite(t *testing.T) {
	suite.Run(t, new(APIUsageTestSuite))
}

func (s *APIUsageTestSuite) SetupSuite() {
	s.schema = makeVirusTotalAPIUsageDataSchema(func(token string) client.Client {
		s.storedTkn = token
		return s.cli
	})
	s.ctx = context.Background()
}

func (s *APIUsageTestSuite) SetupTest() {
	s.cli = &client_mocks.Client{}
}

func (s *APIUsageTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *APIUsageTestSuite) TestSchema() {
	s.Require().NotNil(s.schema)
	s.NotNil(s.schema.Config)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.DataFunc)
}

func (s *APIUsageTestSuite) TestUser() {
	start, err := time.Parse("20060102", "20240101")
	s.Require().NoError(err)
	end, err := time.Parse("20060102", "20240103")
	s.Require().NoError(err)
	s.cli.On("GetUserAPIUsage", mock.Anything, &client.GetUserAPIUsageReq{
		User:      "test_user",
		StartDate: &client.Date{Time: start},
		EndDate:   &client.Date{Time: end},
	}).Return(&client.GetUserAPIUsageRes{
		Data: map[string]any{
			"daily": map[string]any{
				"2024-01-01": map[string]any{},
				"2024-01-02": map[string]any{},
				"2024-01-03": map[string]any{},
			},
		},
	}, nil)
	data, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"api_key": cty.StringVal("test_token"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"user_id":    cty.StringVal("test_user"),
			"group_id":   cty.NullVal(cty.String),
			"start_date": cty.StringVal("20240101"),
			"end_date":   cty.StringVal("20240103"),
		}),
	})
	s.Require().Len(diags, 0)
	s.Equal(plugindata.Map{
		"daily": plugindata.Map{
			"2024-01-01": plugindata.Map{},
			"2024-01-02": plugindata.Map{},
			"2024-01-03": plugindata.Map{},
		},
	}, data)
}

func (s *APIUsageTestSuite) TestGroup() {
	start, err := time.Parse("20060102", "20240101")
	s.Require().NoError(err)
	end, err := time.Parse("20060102", "20240103")
	s.Require().NoError(err)
	s.cli.On("GetGroupAPIUsage", mock.Anything, &client.GetGroupAPIUsageReq{
		Group:     "test_group",
		StartDate: &client.Date{Time: start},
		EndDate:   &client.Date{Time: end},
	}).Return(&client.GetGroupAPIUsageRes{
		Data: map[string]any{
			"daily": map[string]any{
				"2024-01-01": map[string]any{},
				"2024-01-02": map[string]any{},
				"2024-01-03": map[string]any{},
			},
		},
	}, nil)
	data, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"api_key": cty.StringVal("test_token"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"user_id":    cty.NullVal(cty.String),
			"group_id":   cty.StringVal("test_group"),
			"start_date": cty.StringVal("20240101"),
			"end_date":   cty.StringVal("20240103"),
		}),
	})
	s.Require().Len(diags, 0)
	s.Equal(plugindata.Map{
		"daily": plugindata.Map{
			"2024-01-01": plugindata.Map{},
			"2024-01-02": plugindata.Map{},
			"2024-01-03": plugindata.Map{},
		},
	}, data)
}

func (s *APIUsageTestSuite) TestMissingAPIKey() {
	plugintest.NewTestDecoder(s.T(), s.schema.Config).Decode([]diagtest.Assert{
		diagtest.IsError,
	})
}

func (s *APIUsageTestSuite) TestMissingUserIDAndGroupID() {
	data, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"api_key": cty.StringVal("test_token"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"user_id":    cty.NullVal(cty.String),
			"group_id":   cty.NullVal(cty.String),
			"start_date": cty.StringVal("20240101"),
			"end_date":   cty.StringVal("20240103"),
		}),
	})
	s.Require().Len(diags, 1)
	s.Nil(data)
}

func (s *APIUsageTestSuite) TestError() {
	err := errors.New("test error")
	s.cli.On("GetUserAPIUsage", mock.Anything, &client.GetUserAPIUsageReq{
		User: "test_user",
	}).Return(nil, err)
	data, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"api_key": cty.StringVal("test_token"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"user_id":    cty.StringVal("test_user"),
			"group_id":   cty.NullVal(cty.String),
			"start_date": cty.NullVal(cty.String),
			"end_date":   cty.NullVal(cty.String),
		}),
	})
	s.Require().Len(diags, 1)
	s.Nil(data)
}
