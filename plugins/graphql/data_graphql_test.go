// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package graphql

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

type GraphQLDataSourceTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
	plugin *plugin.Schema
}

func (s *GraphQLDataSourceTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.plugin = Plugin("v0.1.0")
}

func (s *GraphQLDataSourceTestSuite) TearDownTest() {
	s.cancel()
}

func TestGraphQLDataSourceSuite(t *testing.T) {
	suite.Run(t, new(GraphQLDataSourceTestSuite))
}

func (s *GraphQLDataSourceTestSuite) TestSchema() {
	source := s.plugin.DataSources["graphql"]
	s.Require().NotNil(source)
	s.NotNil(source.Args)
	s.NotNil(source.DataFunc)
	s.NotNil(source.Config)
}

func (s *GraphQLDataSourceTestSuite) TestBasic() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("application/json", r.Header.Get("Content-Type"))
		s.Equal("application/json", r.Header.Get("Accept"))
		body, err := io.ReadAll(r.Body)
		s.NoError(err)
		s.Equal(`{"query":"query{user{id,name}}"}`, string(body))
		s.Equal("POST", r.Method)
		w.Write([]byte(`{
			"data": {
				"user": {
					"id": "id-1",
					"name": "joe"
				}
			}
		}`))
	}))
	defer srv.Close()
	data, diags := s.plugin.RetrieveData(s.ctx, "graphql", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"url":        cty.StringVal(srv.URL),
			"auth_token": cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"query": cty.StringVal("query{user{id,name}}"),
		}),
	})
	s.Nil(diags)
	s.Equal(plugindata.Map{
		"data": plugindata.Map{
			"user": plugindata.Map{
				"id":   plugindata.String("id-1"),
				"name": plugindata.String("joe"),
			},
		},
	}, data)
}

func (s *GraphQLDataSourceTestSuite) TestWithAuth() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("Bearer token-1", r.Header.Get("Authorization"))
		s.Equal("application/json", r.Header.Get("Content-Type"))
		s.Equal("application/json", r.Header.Get("Accept"))
		body, err := io.ReadAll(r.Body)
		s.NoError(err)
		s.Equal(`{"query":"query{user{id,name}}"}`, string(body))
		s.Equal("POST", r.Method)
		w.Write([]byte(`{
			"data": {
				"user": {
					"id": "id-1",
					"name": "joe"
				}
			}
		}`))
	}))
	defer srv.Close()
	data, diags := s.plugin.RetrieveData(s.ctx, "graphql", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"url":        cty.StringVal(srv.URL),
			"auth_token": cty.StringVal("token-1"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"query": cty.StringVal("query{user{id,name}}"),
		}),
	})
	s.Nil(diags)
	s.Equal(plugindata.Map{
		"data": plugindata.Map{
			"user": plugindata.Map{
				"id":   plugindata.String("id-1"),
				"name": plugindata.String("joe"),
			},
		},
	}, data)
}

func (s *GraphQLDataSourceTestSuite) TestFailRequest() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	data, diags := s.plugin.RetrieveData(s.ctx, "graphql", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"url":        cty.StringVal(srv.URL),
			"auth_token": cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"query": cty.StringVal("query{user{id,name}}"),
		}),
	})
	s.Nil(data)
	s.Len(diags, 1)
	s.Equal("Failed to execute query", diags[0].Summary)
}

func (s *GraphQLDataSourceTestSuite) TestNullURL() {
	data, diags := s.plugin.RetrieveData(s.ctx, "graphql", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"url":        cty.NullVal(cty.String),
			"auth_token": cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"query": cty.StringVal("query{user{id,name}}"),
		}),
	})
	s.Nil(data)
	s.Len(diags, 1)
	s.Equal("Failed to parse config", diags[0].Summary)
}

func (s *GraphQLDataSourceTestSuite) TestEmptyURL() {
	data, diags := s.plugin.RetrieveData(s.ctx, "graphql", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"url":        cty.StringVal(""),
			"auth_token": cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"query": cty.StringVal("query{user{id,name}}"),
		}),
	})
	s.Nil(data)
	s.Len(diags, 1)
	s.Equal("Failed to parse config", diags[0].Summary)
}

func (s *GraphQLDataSourceTestSuite) TestEmptyQuery() {
	data, diags := s.plugin.RetrieveData(s.ctx, "graphql", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"url":        cty.StringVal("http://localhost"),
			"auth_token": cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"query": cty.StringVal(""),
		}),
	})
	s.Nil(data)
	s.Len(diags, 1)
	s.Equal("Failed to parse arguments", diags[0].Summary)
}

func (s *GraphQLDataSourceTestSuite) TestNullQuery() {
	data, diags := s.plugin.RetrieveData(s.ctx, "graphql", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"url":        cty.StringVal("http://localhost"),
			"auth_token": cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"query": cty.NullVal(cty.String),
		}),
	})
	s.Nil(data)
	s.Len(diags, 1)
	s.Equal("Failed to parse arguments", diags[0].Summary)
}

func (s *GraphQLDataSourceTestSuite) TestCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	data, diags := s.plugin.RetrieveData(ctx, "graphql", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"url":        cty.StringVal("http://localhost"),
			"auth_token": cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"query": cty.StringVal("query{user{id,name}}"),
		}),
	})
	s.Nil(data)
	s.Len(diags, 1)
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to execute query",
		Detail:   "failed to execute request: Post \"http://localhost\": context canceled",
	}}, diags)
}
