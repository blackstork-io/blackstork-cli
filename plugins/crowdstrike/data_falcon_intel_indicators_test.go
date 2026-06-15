// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package crowdstrike_test

import (
	"context"
	"errors"
	"testing"

	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client/intel"
	"github.com/crowdstrike/gofalcon/falcon/models"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	mocks "github.com/blackstork-io/blackstork-cli/mocks/plugins/crowdstrike"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugin/plugintest"
	"github.com/blackstork-io/blackstork-cli/plugins/crowdstrike"
)

type CrowdstrikeIntelIndicatorsTestSuite struct {
	suite.Suite
	plugin   *plugin.Schema
	cli      *mocks.Client
	intelCli *mocks.IntelClient
}

func TestCrowdstrikeIntelIndicatorsTestSuite(t *testing.T) {
	suite.Run(t, &CrowdstrikeIntelIndicatorsTestSuite{})
}

func (s *CrowdstrikeIntelIndicatorsTestSuite) SetupSuite() {
	s.plugin = crowdstrike.Plugin("1.2.3", func(cfg *falcon.ApiConfig) (client crowdstrike.Client, err error) {
		return s.cli, nil
	})
}

func (s *CrowdstrikeIntelIndicatorsTestSuite) SetupTest() {
	s.cli = &mocks.Client{}
	s.intelCli = &mocks.IntelClient{}
}

func (s *CrowdstrikeIntelIndicatorsTestSuite) DatasourceNamae() string {
	return "falcon_intel_indicators"
}

func (s *CrowdstrikeIntelIndicatorsTestSuite) Datasource() *plugin.DataSource {
	return s.plugin.DataSources[s.DatasourceNamae()]
}

func (s *CrowdstrikeIntelIndicatorsTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *CrowdstrikeIntelIndicatorsTestSuite) TestSchema() {
	schema := s.plugin.DataSources["falcon_intel_indicators"]
	s.Require().NotNil(schema)
	s.NotNil(schema.Config)
	s.NotNil(schema.Args)
	s.NotNil(schema.DataFunc)
}

func (s *CrowdstrikeIntelIndicatorsTestSuite) String(val string) *string {
	return &val
}

func (s *CrowdstrikeIntelIndicatorsTestSuite) TestBasic() {
	s.cli.On("Intel").Return(s.intelCli)
	s.intelCli.On("QueryIntelIndicatorEntities", mock.MatchedBy(func(params *intel.QueryIntelIndicatorEntitiesParams) bool {
		return params.Limit != nil && *params.Limit == 10
	})).Return(&intel.QueryIntelIndicatorEntitiesOK{
		Payload: &models.DomainPublicIndicatorsV3Response{
			Resources: []*models.DomainPublicIndicatorV3{
				{
					ID:        s.String("test_id_1"),
					Indicator: s.String("test_indicator_1"),
				},
				{
					ID:        s.String("test_id_2"),
					Indicator: s.String("test_indicator_2"),
				},
			},
		},
	}, nil)
	ctx := context.Background()
	data, diags := s.plugin.RetrieveData(ctx, s.DatasourceNamae(), &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.Datasource().Config).
			SetAttr("client_id", cty.StringVal("test")).
			SetAttr("client_secret", cty.StringVal("test")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.Datasource().Args).
			SetAttr("limit", cty.NumberIntVal(10)).
			Decode(),
	})
	s.Require().Nil(diags)
	list := data.AsPluginData().(plugindata.List)
	s.Len(list, 2)
	s.Subset(list[0], plugindata.Map{
		"id":        plugindata.String("test_id_1"),
		"indicator": plugindata.String("test_indicator_1"),
	})
	s.Subset(list[1], plugindata.Map{
		"id":        plugindata.String("test_id_2"),
		"indicator": plugindata.String("test_indicator_2"),
	})
}

func (s *CrowdstrikeIntelIndicatorsTestSuite) TestPayloadErrors() {
	s.cli.On("Intel").Return(s.intelCli)
	s.intelCli.On("QueryIntelIndicatorEntities", mock.MatchedBy(func(params *intel.QueryIntelIndicatorEntitiesParams) bool {
		return params.Limit != nil && *params.Limit == 10
	})).Return(&intel.QueryIntelIndicatorEntitiesOK{
		Payload: &models.DomainPublicIndicatorsV3Response{
			Errors: []*models.MsaAPIError{{
				Message: s.String("something went wrong"),
			}},
		},
	}, nil)
	ctx := context.Background()
	_, diags := s.plugin.RetrieveData(ctx, s.DatasourceNamae(), &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.Datasource().Config).
			SetAttr("client_id", cty.StringVal("test")).
			SetAttr("client_secret", cty.StringVal("test")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.Datasource().Args).
			SetAttr("limit", cty.NumberIntVal(10)).
			Decode(),
	})
	diagtest.Asserts{{
		diagtest.IsError,
		diagtest.SummaryContains("Failed to fetch Falcon Intel Indicators"),
		diagtest.DetailContains("something went wrong"),
	}}.AssertMatch(s.T(), diags, nil)
}

func (s *CrowdstrikeIntelIndicatorsTestSuite) TestError() {
	s.cli.On("Intel").Return(s.intelCli)
	s.intelCli.On("QueryIntelIndicatorEntities", mock.MatchedBy(func(params *intel.QueryIntelIndicatorEntitiesParams) bool {
		return params.Limit != nil && *params.Limit == 10
	})).Return(nil, errors.New("something went wrong"))

	ctx := context.Background()
	_, diags := s.plugin.RetrieveData(ctx, s.DatasourceNamae(), &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.Datasource().Config).
			SetAttr("client_id", cty.StringVal("test")).
			SetAttr("client_secret", cty.StringVal("test")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.Datasource().Args).
			SetAttr("limit", cty.NumberIntVal(10)).
			Decode(),
	})
	diagtest.Asserts{{
		diagtest.IsError,
		diagtest.SummaryContains("Failed to fetch Falcon Intel Indicators"),
		diagtest.DetailContains("something went wrong"),
	}}.AssertMatch(s.T(), diags, nil)
}
