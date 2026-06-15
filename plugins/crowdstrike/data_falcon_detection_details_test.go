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
	"github.com/crowdstrike/gofalcon/falcon/client/detects"
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

type CrowdstrikeDetectionDetailsTestSuite struct {
	suite.Suite
	plugin     *plugin.Schema
	cli        *mocks.Client
	detectsCli *mocks.DetectsClient
}

func TestCrowdstrikeDetectionDetailsTestSuite(t *testing.T) {
	suite.Run(t, &CrowdstrikeDetectionDetailsTestSuite{})
}

func (s *CrowdstrikeDetectionDetailsTestSuite) SetupSuite() {
	s.plugin = crowdstrike.Plugin("1.2.3", func(cfg *falcon.ApiConfig) (client crowdstrike.Client, err error) {
		return s.cli, nil
	})
}

func (s *CrowdstrikeDetectionDetailsTestSuite) DatasourceName() string {
	return "falcon_detection_details"
}

func (s *CrowdstrikeDetectionDetailsTestSuite) Datasource() *plugin.DataSource {
	return s.plugin.DataSources[s.DatasourceName()]
}

func (s *CrowdstrikeDetectionDetailsTestSuite) SetupTest() {
	s.cli = &mocks.Client{}
	s.detectsCli = &mocks.DetectsClient{}
}

func (s *CrowdstrikeDetectionDetailsTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *CrowdstrikeDetectionDetailsTestSuite) TestSchema() {
	schema := s.Datasource()
	s.Require().NotNil(schema)
	s.NotNil(schema.Config)
	s.NotNil(schema.Args)
	s.NotNil(schema.DataFunc)
}

func (s *CrowdstrikeDetectionDetailsTestSuite) String(val string) *string {
	return &val
}

func (s *CrowdstrikeDetectionDetailsTestSuite) TestBasic() {
	s.cli.On("Detects").Return(s.detectsCli)
	s.detectsCli.On("QueryDetects", mock.MatchedBy(func(params *detects.QueryDetectsParams) bool {
		return params.Limit != nil && *params.Limit == 10 && *params.Filter == "test_filter"
	})).Return(&detects.QueryDetectsOK{
		Payload: &models.MsaQueryResponse{
			Resources: []string{"test_resource_1", "test_resource_2"},
		},
	}, nil)
	s.detectsCli.On("GetDetectSummaries", mock.MatchedBy(func(params *detects.GetDetectSummariesParams) bool {
		return params.Body.Ids[0] == "test_resource_1" && params.Body.Ids[1] == "test_resource_2"
	})).Return(&detects.GetDetectSummariesOK{
		Payload: &models.DomainMsaDetectSummariesResponse{
			Resources: []*models.DomainAPIDetectionDocument{
				{
					Cid:    s.String("test_cid_1"),
					Status: s.String("test_status_1"),
				},
				{
					Cid:    s.String("test_cid_1"),
					Status: s.String("test_status_1"),
				},
			},
		},
	}, nil)
	ctx := context.Background()
	data, diags := s.plugin.RetrieveData(ctx, s.DatasourceName(), &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.Datasource().Config).
			SetAttr("client_id", cty.StringVal("test")).
			SetAttr("client_secret", cty.StringVal("test")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.Datasource().Args).
			SetAttr("limit", cty.NumberIntVal(10)).
			SetAttr("filter", cty.StringVal("test_filter")).
			Decode(),
	})
	s.Require().Nil(diags)
	list := data.AsPluginData().(plugindata.List)
	s.Len(list, 2)
	s.Subset(list[0], plugindata.Map{
		"cid":    plugindata.String("test_cid_1"),
		"status": plugindata.String("test_status_1"),
	})
	s.Subset(list[1], plugindata.Map{
		"cid":    plugindata.String("test_cid_1"),
		"status": plugindata.String("test_status_1"),
	})
}

func (s *CrowdstrikeDetectionDetailsTestSuite) TestDetectionError() {
	s.cli.On("Detects").Return(s.detectsCli)
	s.detectsCli.On("QueryDetects", mock.MatchedBy(func(params *detects.QueryDetectsParams) bool {
		return params.Limit != nil && *params.Limit == 10
	})).Return(&detects.QueryDetectsOK{
		Payload: &models.MsaQueryResponse{
			Errors: []*models.MsaAPIError{
				{
					Message: s.String("something went wrong"),
				},
			},
		},
	}, nil)
	ctx := context.Background()
	_, diags := s.plugin.RetrieveData(ctx, s.DatasourceName(), &plugin.RetrieveDataParams{
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
		diagtest.SummaryContains("Failed to query Falcon detects"),
		diagtest.DetailContains("something went wrong"),
	}}.AssertMatch(s.T(), diags, nil)
}

func (s *CrowdstrikeDetectionDetailsTestSuite) TestError() {
	s.cli.On("Detects").Return(s.detectsCli)
	s.detectsCli.On("QueryDetects", mock.MatchedBy(func(params *detects.QueryDetectsParams) bool {
		return params.Limit != nil && *params.Limit == 10
	})).Return(nil, errors.New("something went wrong"))
	ctx := context.Background()
	_, diags := s.plugin.RetrieveData(ctx, s.DatasourceName(), &plugin.RetrieveDataParams{
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
		diagtest.SummaryContains("Failed to query Falcon detects"),
		diagtest.DetailContains("something went wrong"),
	}}.AssertMatch(s.T(), diags, nil)
}
