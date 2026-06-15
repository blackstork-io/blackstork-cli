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

type MicrosoftSecurityDataSourceTestSuite struct {
	suite.Suite
	plugin *plugin.Schema
	schema *plugin.DataSource
	cli    *client_mocks.MicrosoftSecurityClient
}

func TestMicrosoftSecurityDataSourceTestSuite(t *testing.T) {
	suite.Run(t, &MicrosoftSecurityDataSourceTestSuite{})
}

func (s *MicrosoftSecurityDataSourceTestSuite) SetupSuite() {
	s.plugin = microsoft.Plugin(
		"1.0.0",
		nil,
		nil,
		nil,
		func(ctx context.Context, cfg *dataspec.Block) (client microsoft.MicrosoftSecurityClient, err error) {
			return s.cli, nil
		},
	)
	s.schema = s.plugin.DataSources["microsoft_security"]
}

func (s *MicrosoftSecurityDataSourceTestSuite) SetupTest() {
	s.cli = &client_mocks.MicrosoftSecurityClient{}
}

func (s *MicrosoftSecurityDataSourceTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *MicrosoftSecurityDataSourceTestSuite) TestSchema() {
	s.Require().NotNil(s.plugin)
	s.Require().NotNil(s.schema)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.DataFunc)
	s.NotNil(s.schema.Config)
}

func (s *MicrosoftSecurityDataSourceTestSuite) TestBasicObjects() {
	queryParams := url.Values{}
	queryParams.Set("$top", "10")

	size := 1
	endpoint := "/users"

	expectedData := plugindata.List{
		plugindata.Map{
			"foo": plugindata.String("bar"),
		},
	}

	s.cli.On("QueryObjects", mock.Anything, endpoint, queryParams, size).
		Return(expectedData, nil)

	ctx := context.Background()
	result, diags := s.schema.DataFunc(ctx, &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.schema.Config).
			SetAttr("client_id", cty.StringVal("cid")).
			SetAttr("tenant_id", cty.StringVal("tid")).
			SetAttr("client_secret", cty.StringVal("csecret")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("endpoint", cty.StringVal(endpoint)).
			SetAttr("size", cty.NumberIntVal(int64(size))).
			SetAttr("is_object_endpoint", cty.BoolVal(false)).
			SetAttr("query_params", cty.MapVal(map[string]cty.Value{"$top": cty.StringVal("10")})).
			Decode(),
	})
	s.Nil(diags)
	s.NotNil(result)
	s.Equal(expectedData, result.AsPluginData())
}

func (s *MicrosoftSecurityDataSourceTestSuite) TestBasicObject() {
	queryParams := url.Values{}
	queryParams.Set("aaa", "bbb")

	endpoint := "/users"
	expectedData := plugindata.Map{
		"foo": plugindata.String("bar"),
	}

	s.cli.On("QueryObject", mock.Anything, endpoint, queryParams).
		Return(expectedData, nil)

	ctx := context.Background()
	result, diags := s.schema.DataFunc(ctx, &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.schema.Config).
			SetAttr("client_id", cty.StringVal("cid")).
			SetAttr("tenant_id", cty.StringVal("tid")).
			SetAttr("client_secret", cty.StringVal("csecret")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("endpoint", cty.StringVal(endpoint)).
			SetAttr("size", cty.NumberIntVal(int64(999))).
			SetAttr("is_object_endpoint", cty.BoolVal(true)).
			SetAttr("query_params", cty.MapVal(map[string]cty.Value{"aaa": cty.StringVal("bbb")})).
			Decode(),
	})
	s.Nil(diags)
	s.NotNil(result)
	s.Equal(expectedData, result.AsPluginData())
}

func (s *MicrosoftSecurityDataSourceTestSuite) TestClientError() {
	endpoint := "/users"
	size := 1

	queryParams := url.Values{}
	queryParams.Set("$top", "10")

	s.cli.On("QueryObjects", mock.Anything, endpoint, queryParams, size).
		Return(nil, errors.New("dummy error"))

	ctx := context.Background()
	result, diags := s.schema.DataFunc(ctx, &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.schema.Config).
			SetAttr("client_id", cty.StringVal("cid")).
			SetAttr("tenant_id", cty.StringVal("tid")).
			SetAttr("client_secret", cty.StringVal("csecret")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("endpoint", cty.StringVal(endpoint)).
			SetAttr("size", cty.NumberIntVal(int64(size))).
			SetAttr("is_object_endpoint", cty.BoolVal(false)).
			SetAttr("query_params", cty.MapVal(map[string]cty.Value{"$top": cty.StringVal("10")})).
			Decode(),
	})
	s.Nil(result)
	diagtest.Asserts{{
		diagtest.IsError,
		diagtest.DetailContains("dummy error"),
	}}.AssertMatch(s.T(), diags, nil)
}

func (s *MicrosoftSecurityDataSourceTestSuite) TestMissingArgs() {
	plugintest.NewTestDecoder(
		s.T(),
		s.schema.Args,
	).Decode([]diagtest.Assert{
		diagtest.IsError,
		diagtest.SummaryEquals("Missing required attribute"),
		diagtest.DetailContains("endpoint"),
	})
}

func (s *MicrosoftSecurityDataSourceTestSuite) TestMissingConfig() {
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

func (s *MicrosoftSecurityDataSourceTestSuite) TestMissingCredentials() {
	s.plugin = microsoft.Plugin(
		"1.0.0",
		nil,
		nil,
		nil,
		microsoft.MakeDefaultMicrosoftSecurityClientLoader(client.AcquireAzureToken),
	)
	s.schema = s.plugin.DataSources["microsoft_security"]

	ctx := context.Background()
	result, diags := s.schema.DataFunc(ctx, &plugin.RetrieveDataParams{
		Config: plugintest.NewTestDecoder(s.T(), s.schema.Config).
			SetAttr("client_id", cty.StringVal("cid")).
			SetAttr("tenant_id", cty.StringVal("tid")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("endpoint", cty.StringVal("/foo")).
			SetAttr("size", cty.NumberIntVal(12)).
			SetAttr("is_object_endpoint", cty.BoolVal(false)).
			SetAttr("query_params", cty.MapVal(map[string]cty.Value{"$top": cty.StringVal("10")})).
			Decode(),
	})
	s.Nil(result)
	diagtest.Asserts{{
		diagtest.IsError,
		diagtest.DetailContains("Either `client_secret` or `private_key` / `private_key_file` arguments must be provide"),
	}}.AssertMatch(s.T(), diags, nil)
}
