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
	"net/url"
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

type MicrosoftGraphDataSourceTestSuite struct {
	suite.Suite
	plugin *plugin.Schema
	schema *plugin.DataSource
	cli    *client_mocks.MicrosoftGraphClient
}

func TestMicrosoftGraphDataSourceTestSuite(t *testing.T) {
	suite.Run(t, &MicrosoftGraphDataSourceTestSuite{})
}

func (s *MicrosoftGraphDataSourceTestSuite) SetupSuite() {
	s.plugin = microsoft.Plugin(
		"1.0.0",
		nil,
		nil,
		func(ctx context.Context, apiVersion string, cfg *dataspec.Block) (client microsoft.MicrosoftGraphClient, err error) {
			return s.cli, nil
		},
		nil,
	)
	s.schema = s.plugin.DataSources["microsoft_graph"]
}

func (s *MicrosoftGraphDataSourceTestSuite) SetupTest() {
	s.cli = &client_mocks.MicrosoftGraphClient{}
}

func (s *MicrosoftGraphDataSourceTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *MicrosoftGraphDataSourceTestSuite) TestSchema() {
	s.Require().NotNil(s.plugin)
	s.Require().NotNil(s.schema)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.DataFunc)
	s.NotNil(s.schema.Config)
}

func (s *MicrosoftGraphDataSourceTestSuite) TestBasic() {
	expectedData := plugindata.List{
		plugindata.Map{
			"severity":    plugindata.String("High"),
			"displayName": plugindata.String("Incident 1"),
		},
	}
	s.cli.On("QueryObjects", mock.Anything, "/security/incidents", url.Values{"$top": []string{"10"}}, 1).
		Return(expectedData, nil)
	ctx := context.Background()
	result, diags := s.schema.DataFunc(ctx, &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.schema.Config).
			SetAttr("client_id", cty.StringVal("cid")).
			SetAttr("tenant_id", cty.StringVal("tid")).
			SetAttr("client_secret", cty.StringVal("csecret")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("endpoint", cty.StringVal("/security/incidents")).
			SetAttr("api_version", cty.StringVal("v1")).
			SetAttr("size", cty.NumberIntVal(1)).
			SetAttr("query_params", cty.MapVal(map[string]cty.Value{"$top": cty.StringVal("10")})).
			Decode(),
	})
	s.Nil(diags)
	s.Equal(expectedData, result.AsPluginData())
}

func (s *MicrosoftGraphDataSourceTestSuite) TestBasicObject() {
	expectedData := plugindata.Map{
		"value": plugindata.Map{
			"severity":    plugindata.String("High"),
			"displayName": plugindata.String("Incident 1"),
		},
	}
	s.cli.On("QueryObject", mock.Anything, "/security/incidents/123", url.Values{}).Return(expectedData, nil)
	ctx := context.Background()
	result, diags := s.schema.DataFunc(ctx, &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.schema.Config).
			SetAttr("client_id", cty.StringVal("cid")).
			SetAttr("tenant_id", cty.StringVal("tid")).
			SetAttr("client_secret", cty.StringVal("csecret")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("endpoint", cty.StringVal("/security/incidents/123")).
			SetAttr("is_object_endpoint", cty.StringVal("true")).
			Decode(),
	})
	s.Nil(diags)
	s.Equal(expectedData, result.AsPluginData())
}

func (s *MicrosoftGraphDataSourceTestSuite) TestClientError() {
	s.cli.On("QueryObjects", mock.Anything, "/security/incidents", url.Values{"$top": []string{"10"}}, 1).
		Return(nil, errors.New("microsoft graph client returned status code: 400"))
	ctx := context.Background()
	result, diags := s.schema.DataFunc(ctx, &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.schema.Config).
			SetAttr("client_id", cty.StringVal("cid")).
			SetAttr("tenant_id", cty.StringVal("tid")).
			SetAttr("client_secret", cty.StringVal("csecret")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("endpoint", cty.StringVal("/security/incidents")).
			SetAttr("api_version", cty.StringVal("v1")).
			SetAttr("size", cty.NumberIntVal(1)).
			SetAttr("query_params", cty.MapVal(map[string]cty.Value{"$top": cty.StringVal("10")})).
			Decode(),
	})
	s.Nil(result)
	diagtest.Asserts{{
		diagtest.IsError,
		diagtest.DetailContains("microsoft graph client returned status code: 400"),
	}}.AssertMatch(s.T(), diags, nil)
}

func (s *MicrosoftGraphDataSourceTestSuite) TestMissingArgs() {
	plugintest.NewTestDecoder(
		s.T(),
		s.schema.Args,
	).Decode([]diagtest.Assert{
		diagtest.IsError,
		diagtest.SummaryEquals("Missing required attribute"),
		diagtest.DetailContains("endpoint"),
	})
}

func (s *MicrosoftGraphDataSourceTestSuite) TestMissingConfig() {
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

func (s *MicrosoftGraphDataSourceTestSuite) TestMissingCredentials() {
	s.plugin = microsoft.Plugin(
		"1.0.0",
		nil,
		nil,
		microsoft.MakeDefaultMicrosoftGraphClientLoader(client.AcquireAzureToken),
		nil,
	)
	s.schema = s.plugin.DataSources["microsoft_graph"]

	ctx := context.Background()
	result, diags := s.schema.DataFunc(ctx, &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.schema.Config).
			SetAttr("client_id", cty.StringVal("cid")).
			SetAttr("tenant_id", cty.StringVal("tid")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("endpoint", cty.StringVal("/security/incidents")).
			SetAttr("api_version", cty.StringVal("v1")).
			SetAttr("query_params", cty.MapVal(map[string]cty.Value{"$top": cty.StringVal("10")})).
			SetAttr("size", cty.NumberIntVal(1)).
			Decode(),
	})
	s.Nil(result)
	diagtest.Asserts{{
		diagtest.IsError,
		diagtest.DetailContains("Either `client_secret` or `private_key` / `private_key_file` arguments must be provide"),
	}}.AssertMatch(s.T(), diags, nil)
}
