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
	"github.com/crowdstrike/gofalcon/falcon/client/cspm_registration"
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

type CrowdstrikeCspmIomsTestSuite struct {
	suite.Suite
	plugin  *plugin.Schema
	cli     *mocks.Client
	cspmCli *mocks.CspmRegistrationClient
}

func TestCrowdstrikeCspmIomsSuite(t *testing.T) {
	suite.Run(t, &CrowdstrikeCspmIomsTestSuite{})
}

func (s *CrowdstrikeCspmIomsTestSuite) SetupSuite() {
	s.plugin = crowdstrike.Plugin("1.2.3", func(cfg *falcon.ApiConfig) (client crowdstrike.Client, err error) {
		return s.cli, nil
	})
}

func (s *CrowdstrikeCspmIomsTestSuite) SetupTest() {
	s.cli = &mocks.Client{}
	s.cspmCli = &mocks.CspmRegistrationClient{}
}

func (s *CrowdstrikeCspmIomsTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *CrowdstrikeCspmIomsTestSuite) TestSchema() {
	schema := s.plugin.DataSources["falcon_cspm_ioms"]
	s.Require().NotNil(schema)
	s.NotNil(schema.Config)
	s.NotNil(schema.Args)
	s.NotNil(schema.DataFunc)
}

func (s *CrowdstrikeCspmIomsTestSuite) String(val string) *string {
	return &val
}

func (s *CrowdstrikeCspmIomsTestSuite) TestBasic() {
	s.cli.On("CspmRegistration").Return(s.cspmCli)
	s.cspmCli.On("GetConfigurationDetections", mock.MatchedBy(func(params *cspm_registration.GetConfigurationDetectionsParams) bool {
		return params.Limit != nil && *params.Limit == 10
	})).Return(&cspm_registration.GetConfigurationDetectionsOK{
		Payload: &models.RegistrationExternalIOMEventResponse{
			Resources: &models.RegistrationIOMResources{
				Events: []*models.RegistrationIOMEvent{
					{
						AccountID:     s.String("test_account_id_1"),
						AccountName:   s.String("test_account_name_1"),
						AzureTenantID: s.String("test_azure_tenant_id_1"),
						Cid:           s.String("test_cid_1"),
						CloudProvider: s.String("test_cloud_provider_1"),
					},
					{
						AccountID:     s.String("test_account_id_2"),
						AccountName:   s.String("test_account_name_2"),
						AzureTenantID: s.String("test_azure_tenant_id_2"),
						Cid:           s.String("test_cid_2"),
						CloudProvider: s.String("test_cloud_provider_2"),
					},
				},
			},
		},
	}, nil)
	ctx := context.Background()
	data, diags := s.plugin.RetrieveData(ctx, "falcon_cspm_ioms", &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["falcon_cspm_ioms"].Config).
			SetAttr("client_id", cty.StringVal("test")).
			SetAttr("client_secret", cty.StringVal("test")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["falcon_cspm_ioms"].Args).
			SetAttr("limit", cty.NumberIntVal(10)).
			Decode(),
	})
	s.Require().Nil(diags)
	list := data.AsPluginData().(plugindata.List)
	s.Len(list, 2)
	s.Subset(list[0], plugindata.Map{
		"account_id":      plugindata.String("test_account_id_1"),
		"account_name":    plugindata.String("test_account_name_1"),
		"azure_tenant_id": plugindata.String("test_azure_tenant_id_1"),
		"cid":             plugindata.String("test_cid_1"),
		"cloud_provider":  plugindata.String("test_cloud_provider_1"),
	})
	s.Subset(list[1], plugindata.Map{
		"account_id":      plugindata.String("test_account_id_2"),
		"account_name":    plugindata.String("test_account_name_2"),
		"azure_tenant_id": plugindata.String("test_azure_tenant_id_2"),
		"cid":             plugindata.String("test_cid_2"),
		"cloud_provider":  plugindata.String("test_cloud_provider_2"),
	})
}

func (s *CrowdstrikeCspmIomsTestSuite) TestPayloadErrors() {
	s.cli.On("CspmRegistration").Return(s.cspmCli)
	s.cspmCli.On("GetConfigurationDetections", mock.MatchedBy(func(params *cspm_registration.GetConfigurationDetectionsParams) bool {
		return params.Limit != nil && *params.Limit == 10
	})).Return(&cspm_registration.GetConfigurationDetectionsOK{
		Payload: &models.RegistrationExternalIOMEventResponse{
			Errors: []*models.MsaAPIError{{
				Message: s.String("something went wrong"),
			}},
		},
	}, nil)
	ctx := context.Background()
	_, diags := s.plugin.RetrieveData(ctx, "falcon_cspm_ioms", &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["falcon_cspm_ioms"].Config).
			SetAttr("client_id", cty.StringVal("test")).
			SetAttr("client_secret", cty.StringVal("test")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["falcon_cspm_ioms"].Args).
			SetAttr("limit", cty.NumberIntVal(10)).
			Decode(),
	})
	diagtest.Asserts{{
		diagtest.IsError,
		diagtest.SummaryContains("Failed to fetch Falcon CSPM IOMs"),
		diagtest.DetailContains("something went wrong"),
	}}.AssertMatch(s.T(), diags, nil)
}

func (s *CrowdstrikeCspmIomsTestSuite) TestError() {
	s.cli.On("CspmRegistration").Return(s.cspmCli)
	s.cspmCli.On("GetConfigurationDetections", mock.MatchedBy(func(params *cspm_registration.GetConfigurationDetectionsParams) bool {
		return params.Limit != nil && *params.Limit == 10
	})).Return(nil, errors.New("something went wrong"))
	ctx := context.Background()
	_, diags := s.plugin.RetrieveData(ctx, "falcon_cspm_ioms", &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["falcon_cspm_ioms"].Config).
			SetAttr("client_id", cty.StringVal("test")).
			SetAttr("client_secret", cty.StringVal("test")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.plugin.DataSources["falcon_cspm_ioms"].Args).
			SetAttr("limit", cty.NumberIntVal(10)).
			Decode(),
	})
	diagtest.Asserts{{
		diagtest.IsError,
		diagtest.SummaryContains("Failed to fetch Falcon CSPM IOMs"),
		diagtest.DetailContains("something went wrong"),
	}}.AssertMatch(s.T(), diags, nil)
}
