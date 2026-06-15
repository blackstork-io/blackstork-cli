// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package microsoft

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	client_mocks "github.com/blackstork-io/blackstork-cli/mocks/plugins/microsoft"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

type SentinelIncidentsDataSourceTestSuite struct {
	suite.Suite
	schema *plugin.DataSource
	ctx    context.Context
	cli    *client_mocks.AzureClient
}

func TestSentinelIncidentsDataSourceTestSuite(t *testing.T) {
	suite.Run(t, new(SentinelIncidentsDataSourceTestSuite))
}

func (s *SentinelIncidentsDataSourceTestSuite) SetupSuite() {
	s.schema = makeMicrosoftSentinelIncidentsDataSource(func(ctx context.Context, cfg *dataspec.Block) (AzureClient, error) {
		return s.cli, nil
	})
	s.ctx = context.Background()
}

func (s *SentinelIncidentsDataSourceTestSuite) SetupTest() {
	s.cli = &client_mocks.AzureClient{}
}

func (s *SentinelIncidentsDataSourceTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *SentinelIncidentsDataSourceTestSuite) TestSchema() {
	s.Require().NotNil(s.schema)
	s.NotNil(s.schema.Config)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.DataFunc)
}

func (s *SentinelIncidentsDataSourceTestSuite) TestSize() {
	endpoint := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.OperationalInsights/workspaces/%s/providers/Microsoft.SecurityInsights/incidents",
		"test_subscription_id",
		"test_resource_group_name",
		"test_workspace_name",
	)
	size := 123

	s.cli.On("QueryObjects", mock.Anything, endpoint, url.Values{}, size).Return(plugindata.List{
		plugindata.Map{
			"any": plugindata.String("data"),
		},
	}, nil)
	res, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"tenant_id":           cty.StringVal("test_tenant_id"),
			"client_id":           cty.StringVal("test_client_id"),
			"client_secret":       cty.StringVal("test_client_secret"),
			"subscription_id":     cty.StringVal("test_subscription_id"),
			"resource_group_name": cty.StringVal("test_resource_group_name"),
			"workspace_name":      cty.StringVal("test_workspace_name"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"size": cty.NumberIntVal(int64(size)),
		}),
	})
	s.Len(diags, 0)
	s.Equal(plugindata.List{
		plugindata.Map{
			"any": plugindata.String("data"),
		},
	}, res)
}

func (s *SentinelIncidentsDataSourceTestSuite) TestFull() {
	endpoint := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.OperationalInsights/workspaces/%s/providers/Microsoft.SecurityInsights/incidents",
		"test_subscription_id",
		"test_resource_group_name",
		"test_workspace_name",
	)
	query := url.Values{}
	query.Set("$filter", "test_filter")
	query.Set("$orderby", "test_order_by")

	s.cli.On("QueryObjects", mock.Anything, endpoint, query, 10).Return(plugindata.List{
		plugindata.Map{
			"any": plugindata.String("data"),
		},
	}, nil)

	res, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"tenant_id":           cty.StringVal("test_tenant_id"),
			"client_id":           cty.StringVal("test_client_id"),
			"client_secret":       cty.StringVal("test_client_secret"),
			"subscription_id":     cty.StringVal("test_subscription_id"),
			"resource_group_name": cty.StringVal("test_resource_group_name"),
			"workspace_name":      cty.StringVal("test_workspace_name"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"filter":   cty.StringVal("test_filter"),
			"order_by": cty.StringVal("test_order_by"),
			"size":     cty.NumberIntVal(int64(10)),
		}),
	})
	s.Len(diags, 0)
	s.Equal(plugindata.List{
		plugindata.Map{
			"any": plugindata.String("data"),
		},
	}, res)
}

func (s *SentinelIncidentsDataSourceTestSuite) TestError() {
	errTest := fmt.Errorf("test_error")

	s.cli.On("QueryObjects", mock.Anything, mock.Anything, mock.Anything, 10).Return(nil, errTest)

	_, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"tenant_id":           cty.StringVal("test_tenant_id"),
			"client_id":           cty.StringVal("test_client_id"),
			"client_secret":       cty.StringVal("test_client_secret"),
			"subscription_id":     cty.StringVal("test_subscription_id"),
			"resource_group_name": cty.StringVal("test_resource_group_name"),
			"workspace_name":      cty.StringVal("test_workspace_name"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"size": cty.NumberIntVal(10),
		}),
	})
	s.Len(diags, 1)
	s.Equal("Unable to get Microsoft Sentinel incidents", diags[0].Summary)
	s.Equal(errTest.Error(), diags[0].Detail)
}
