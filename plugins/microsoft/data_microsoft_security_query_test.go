// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package microsoft_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	client_mocks "github.com/blackstork-io/blackstork-cli/mocks/plugins/microsoft"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugin/plugintest"
	"github.com/blackstork-io/blackstork-cli/plugins/microsoft"
	"github.com/blackstork-io/blackstork-cli/plugins/microsoft/client"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

type MicrosoftSecurityQueryDataSourceTestSuite struct {
	suite.Suite
	plugin *plugin.Schema
	schema *plugin.DataSource
	cli    *client_mocks.MicrosoftSecurityClient
}

func TestMicrosoftSecurityQueryDataSourceTestSuite(t *testing.T) {
	suite.Run(t, &MicrosoftSecurityQueryDataSourceTestSuite{})
}

func (s *MicrosoftSecurityQueryDataSourceTestSuite) SetupSuite() {
	s.plugin = microsoft.Plugin(
		"1.0.0",
		nil,
		nil,
		nil,
		func(ctx context.Context, cfg *dataspec.Block) (client microsoft.MicrosoftSecurityClient, err error) {
			return s.cli, nil
		},
	)
	s.schema = s.plugin.DataSources["microsoft_security_query"]
}

func (s *MicrosoftSecurityQueryDataSourceTestSuite) SetupTest() {
	s.cli = &client_mocks.MicrosoftSecurityClient{}
}

func (s *MicrosoftSecurityQueryDataSourceTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *MicrosoftSecurityQueryDataSourceTestSuite) TestSchema() {
	s.Require().NotNil(s.plugin)
	s.Require().NotNil(s.schema)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.DataFunc)
	s.NotNil(s.schema.Config)
}

func (s *MicrosoftSecurityQueryDataSourceTestSuite) TestBasic() {
	testQuery := "test-query"
	expectedData := plugindata.List{
		plugindata.Map{
			"foo": plugindata.String("bar"),
		},
	}

	rawResponse := plugindata.Map{
		"Results": plugindata.List{
			plugindata.Map{
				"foo": plugindata.String("bar"),
			},
		},
	}
	s.cli.On("RunAdvancedQuery", mock.Anything, testQuery).Return(rawResponse, nil)
	ctx := context.Background()
	result, diags := s.schema.DataFunc(ctx, &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.schema.Config).
			SetAttr("client_id", cty.StringVal("cid")).
			SetAttr("tenant_id", cty.StringVal("tid")).
			SetAttr("client_secret", cty.StringVal("csecret")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("query", cty.StringVal(testQuery)).
			Decode(),
	})
	s.Nil(diags)
	s.NotNil(result)
	s.Equal(expectedData, result.AsPluginData())
}

func (s *MicrosoftSecurityQueryDataSourceTestSuite) TestClientError() {
	testQuery := "test-query"
	s.cli.On("RunAdvancedQuery", mock.Anything, testQuery).
		Return(nil, errors.New("Microsoft Security query API returned status code: 400"))

	ctx := context.Background()
	result, diags := s.schema.DataFunc(ctx, &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.schema.Config).
			SetAttr("client_id", cty.StringVal("cid")).
			SetAttr("tenant_id", cty.StringVal("tid")).
			SetAttr("client_secret", cty.StringVal("csecret")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("query", cty.StringVal(testQuery)).
			Decode(),
	})
	s.Nil(result)
	diagtest.Asserts{{
		diagtest.IsError,
		diagtest.DetailContains("Microsoft Security query API returned status code: 400"),
	}}.AssertMatch(s.T(), diags, nil)
}

func (s *MicrosoftSecurityQueryDataSourceTestSuite) TestMissingArgs() {
	plugintest.NewTestDecoder(
		s.T(),
		s.schema.Args,
	).Decode([]diagtest.Assert{
		diagtest.IsError,
		diagtest.SummaryEquals("Missing required attribute"),
		diagtest.DetailContains("query"),
	})
}

func (s *MicrosoftSecurityQueryDataSourceTestSuite) TestMissingConfig() {
	plugintest.NewTestDecoder(
		s.T(),
		s.schema.Config,
	).Decode([]diagtest.Assert{
		diagtest.IsError,
		diagtest.SummaryEquals("Missing required attribute"),
		diagtest.DetailContains("client_id"),
	}, []diagtest.Assert{
		diagtest.IsError,
		diagtest.SummaryEquals("Missing required attribute"),
		diagtest.DetailContains("tenant_id"),
	})
}

func (s *MicrosoftSecurityQueryDataSourceTestSuite) TestMissingCredentials() {
	s.plugin = microsoft.Plugin(
		"1.0.0",
		nil,
		nil,
		nil,
		microsoft.MakeDefaultMicrosoftSecurityClientLoader(client.AcquireAzureToken),
	)
	s.schema = s.plugin.DataSources["microsoft_security_query"]

	ctx := context.Background()
	result, diags := s.schema.DataFunc(ctx, &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.schema.Config).
			SetAttr("client_id", cty.StringVal("cid")).
			SetAttr("tenant_id", cty.StringVal("tid")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("query", cty.StringVal("some")).
			Decode(),
	})
	s.Nil(result)
	diagtest.Asserts{{
		diagtest.IsError,
		diagtest.DetailContains("Either `client_secret` or `private_key` / `private_key_file` arguments must be provide"),
	}}.AssertMatch(s.T(), diags, nil)
}
