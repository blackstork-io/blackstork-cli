// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package opencti

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

type OpenCTIDataSourceTestSuite struct {
	suite.Suite
	plugin *plugin.Schema
}

func TestOpenCTIDataSourceTestSuite(t *testing.T) {
	suite.Run(t, new(OpenCTIDataSourceTestSuite))
}

func (s *OpenCTIDataSourceTestSuite) SetupTest() {
	s.plugin = Plugin("1.2.3")
}

func (s *OpenCTIDataSourceTestSuite) TestSchema() {
	source := s.plugin.DataSources["opencti"]
	s.Require().NotNil(source)
	s.NotNil(source.Config)
	s.NotNil(source.Args)
	s.NotNil(source.DataFunc)
}

func (s *OpenCTIDataSourceTestSuite) TestBasicValid() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("application/json", r.Header.Get("Content-Type"))
		s.Equal("application/json", r.Header.Get("Accept"))
		body, err := io.ReadAll(r.Body)
		s.NoError(err)
		s.Equal(`{"query":"query issue { stixCoreRelationships { edges { node { x_opencti_stix_ids } } } }"}`, string(body))
		s.Equal("POST", r.Method)
		w.Write([]byte(`
					{
						"data": {
							"stixCoreRelationships": {
								"edges": []
							}
						}
					}
				`))
	}))
	defer srv.Close()
	ctx := context.Background()
	data, diags := s.plugin.RetrieveData(ctx, "opencti", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"graphql_url": cty.StringVal(srv.URL),
			"auth_token":  cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"graphql_query": cty.StringVal("query issue { stixCoreRelationships { edges { node { x_opencti_stix_ids } } } }"),
		}),
	})
	s.Nil(diags)
	s.Equal(plugindata.Map{
		"data": plugindata.Map{
			"stixCoreRelationships": plugindata.Map{
				"edges": plugindata.List{},
			},
		},
	}, data)
}

func (s *OpenCTIDataSourceTestSuite) TestFailRequest() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	ctx := context.Background()
	data, diags := s.plugin.RetrieveData(ctx, "opencti", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"graphql_url": cty.StringVal(srv.URL),
			"auth_token":  cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"graphql_query": cty.StringVal("query issue { stixCoreRelationships { edges { node { x_opencti_stix_ids } } } }"),
		}),
	})
	s.Nil(data)
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to execute query",
		Detail:   "unexpected status code: 404",
	}}, diags)
}

func (s *OpenCTIDataSourceTestSuite) TestInvalidQuery() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("application/json", r.Header.Get("Content-Type"))
		s.Equal("application/json", r.Header.Get("Accept"))
		body, err := io.ReadAll(r.Body)
		s.NoError(err)
		s.Equal(`{"query":"query issue { stixCoreRelationshipsInvalid { edges { node { x_opencti_stix_ids } } } }"}`, string(body))
		s.Equal("POST", r.Method)
		w.Write([]byte(`
					{
						"data": {
							"stixCoreRelationships": {
								"edges": []
							}
						}
					}
				`))
	}))
	defer srv.Close()
	ctx := context.Background()
	data, diags := s.plugin.RetrieveData(ctx, "opencti", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"graphql_url": cty.StringVal(srv.URL),
			"auth_token":  cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"graphql_query": cty.StringVal("query issue { stixCoreRelationshipsInvalid { edges { node { x_opencti_stix_ids } } } }"),
		}),
	})
	s.Nil(data)
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Invalid GraphQL query",
		Detail:   "external: field: stixCoreRelationshipsInvalid not defined on type: Query, locations: [], path: [query]",
	}}, diags)
}

func (s *OpenCTIDataSourceTestSuite) TestWithAuth() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("Bearer token-123", r.Header.Get("Authorization"))
		s.Equal("application/json", r.Header.Get("Content-Type"))
		s.Equal("application/json", r.Header.Get("Accept"))
		body, err := io.ReadAll(r.Body)
		s.NoError(err)
		s.Equal(`{"query":"query issue { stixCoreRelationships { edges { node { x_opencti_stix_ids } } } }"}`, string(body))
		s.Equal("POST", r.Method)
		w.Write([]byte(`
					{
						"data": {
							"stixCoreRelationships": {
								"edges": []
							}
						}
					}
				`))
	}))
	defer srv.Close()
	ctx := context.Background()
	data, diags := s.plugin.RetrieveData(ctx, "opencti", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"graphql_url": cty.StringVal(srv.URL),
			"auth_token":  cty.StringVal("token-123"),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"graphql_query": cty.StringVal("query issue { stixCoreRelationships { edges { node { x_opencti_stix_ids } } } }"),
		}),
	})
	s.Nil(diags)
	s.Equal(plugindata.Map{
		"data": plugindata.Map{
			"stixCoreRelationships": plugindata.Map{
				"edges": plugindata.List{},
			},
		},
	}, data)
}

func (s *OpenCTIDataSourceTestSuite) TestNullURL() {
	ctx := context.Background()
	data, diags := s.plugin.RetrieveData(ctx, "opencti", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"graphql_url": cty.NullVal(cty.String),
			"auth_token":  cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"graphql_query": cty.StringVal("query{user{id,name}}"),
		}),
	})
	s.Nil(data)
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to parse config",
		Detail:   "graphql_url is required",
	}}, diags)
}

func (s *OpenCTIDataSourceTestSuite) TestEmptyURL() {
	ctx := context.Background()
	data, diags := s.plugin.RetrieveData(ctx, "opencti", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"graphql_url": cty.StringVal(""),
			"auth_token":  cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"graphql_query": cty.StringVal("query{user{id,name}}"),
		}),
	})
	s.Nil(data)
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to parse config",
		Detail:   "graphql_url is required",
	}}, diags)
}

func (s *OpenCTIDataSourceTestSuite) TestEmptyQuery() {
	ctx := context.Background()
	data, diags := s.plugin.RetrieveData(ctx, "opencti", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"graphql_url": cty.StringVal("http://localhost"),
			"auth_token":  cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"graphql_query": cty.StringVal(""),
		}),
	})
	s.Nil(data)
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to parse arguments",
		Detail:   "graphql_query is required",
	}}, diags)
}

func (s *OpenCTIDataSourceTestSuite) TestNullQuery() {
	ctx := context.Background()
	data, diags := s.plugin.RetrieveData(ctx, "opencti", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"graphql_url": cty.StringVal("http://localhost"),
			"auth_token":  cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"graphql_query": cty.NullVal(cty.String),
		}),
	})
	s.Nil(data)
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to parse arguments",
		Detail:   "graphql_query is required",
	}}, diags)
}

func (s *OpenCTIDataSourceTestSuite) TestCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	data, diags := s.plugin.RetrieveData(ctx, "opencti", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"graphql_url": cty.StringVal("http://localhost"),
			"auth_token":  cty.NullVal(cty.String),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"graphql_query": cty.StringVal("query issue { stixCoreRelationships { edges { node { x_opencti_stix_ids } } } }"),
		}),
	})
	s.Nil(data)
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to execute query",
		Detail:   "Post \"http://localhost\": context canceled",
	}}, diags)
}

// Old tests from that need to be updated

// func (s *OpenCTIDataSourceTestSuite) TestBasicValid() {
// 	want := plugininterface.Result{
// 		Result: jsonAny(`
// 			{
// 				"data": {
// 					"stixCoreRelationships": {
// 						"edges": []
// 					}
// 				}
// 			}
// 		`),
// 	}
// 	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		s.Equal("application/json", r.Header.Get("Content-Type"))
// 		s.Equal("application/json", r.Header.Get("Accept"))
// 		body, err := io.ReadAll(r.Body)
// 		s.NoError(err)
// 		s.Equal(`{"query":"query issue { stixCoreRelationships { edges { node { x_opencti_stix_ids } } } }"}`, string(body))
// 		s.Equal("POST", r.Method)
// 		w.Write([]byte(`
// 			{
// 				"data": {
// 					"stixCoreRelationships": {
// 						"edges": []
// 					}
// 				}
// 			}
// 		`))
// 	}))
// 	defer srv.Close()
// 	p := Plugin{}
// 	result := p.Call(plugininterface.Args{
// 		Kind: "data",
// 		Name: "opencti",
// 		Config: cty.ObjectVal(map[string]cty.Value{
// 			"graphql_url": cty.StringVal(srv.URL),
// 			"auth_token":  cty.NullVal(cty.String),
// 		}),
// 		Args: cty.ObjectVal(map[string]cty.Value{
// 			"graphql_query": cty.StringVal("query issue { stixCoreRelationships { edges { node { x_opencti_stix_ids } } } }"),
// 		}),
// 	})
// 	s.Equal(want, result)
// }

// func (s *OpenCTIDataSourceTestSuite) TestFailRequest() {
// 	want := plugininterface.Result{
// 		Diags: diagnostics.Diag{
// 			{
// 				Severity: hcl.DiagError,
// 				Summary:  "Failed to execute query",
// 				Detail:   "unexpected status code: 404",
// 			},
// 		},
// 	}
// 	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		w.WriteHeader(http.StatusNotFound)
// 	}))
// 	defer srv.Close()
// 	p := Plugin{}
// 	result := p.Call(plugininterface.Args{
// 		Kind: "data",
// 		Name: "opencti",
// 		Config: cty.ObjectVal(map[string]cty.Value{
// 			"graphql_url": cty.StringVal(srv.URL),
// 			"auth_token":  cty.NullVal(cty.String),
// 		}),
// 		Args: cty.ObjectVal(map[string]cty.Value{
// 			"graphql_query": cty.StringVal("query issue { stixCoreRelationships { edges { node { x_opencti_stix_ids } } } }"),
// 		}),
// 	})
// 	s.Equal(want, result)
// }

// func (s *OpenCTIDataSourceTestSuite) TestInvalidQuery() {
// 	want := plugininterface.Result{
// 		Diags: diagnostics.Diag{
// 			{
// 				Severity: hcl.DiagError,
// 				Summary:  "Invalid GraphQL query",
// 				Detail:   "external: field: stixCoreRelationshipsInvalid not defined on type: Query, locations: [], path: [query]",
// 			},
// 		},
// 	}
// 	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		s.Equal("application/json", r.Header.Get("Content-Type"))
// 		s.Equal("application/json", r.Header.Get("Accept"))
// 		body, err := io.ReadAll(r.Body)
// 		s.NoError(err)
// 		s.Equal(`{"query":"query issue { stixCoreRelationshipsInvalid { edges { node { x_opencti_stix_ids } } } }"}`, string(body))
// 		s.Equal("POST", r.Method)
// 		w.Write([]byte(`
// 			{
// 				"data": {
// 					"stixCoreRelationships": {
// 						"edges": []
// 					}
// 				}
// 			}
// 		`))
// 	}))
// 	defer srv.Close()
// 	p := Plugin{}
// 	result := p.Call(plugininterface.Args{
// 		Kind: "data",
// 		Name: "opencti",
// 		Config: cty.ObjectVal(map[string]cty.Value{
// 			"graphql_url": cty.StringVal(srv.URL),
// 			"auth_token":  cty.NullVal(cty.String),
// 		}),
// 		Args: cty.ObjectVal(map[string]cty.Value{
// 			"graphql_query": cty.StringVal("query issue { stixCoreRelationshipsInvalid { edges { node { x_opencti_stix_ids } } } }"),
// 		}),
// 	})
// 	s.Equal(want, result)
// }

// func (s *OpenCTIDataSourceTestSuite) TestWithAuth() {
// 	want := plugininterface.Result{
// 		Result: jsonAny(`
// 			{
// 				"data": {
// 					"stixCoreRelationships": {
// 						"edges": []
// 					}
// 				}
// 			}
// 		`),
// 	}
// 	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		s.Equal("Bearer token-123", r.Header.Get("Authorization"))
// 		s.Equal("application/json", r.Header.Get("Content-Type"))
// 		s.Equal("application/json", r.Header.Get("Accept"))
// 		body, err := io.ReadAll(r.Body)
// 		s.NoError(err)
// 		s.Equal(`{"query":"query issue { stixCoreRelationships { edges { node { x_opencti_stix_ids } } } }"}`, string(body))
// 		s.Equal("POST", r.Method)
// 		w.Write([]byte(`
// 			{
// 				"data": {
// 					"stixCoreRelationships": {
// 						"edges": []
// 					}
// 				}
// 			}
// 		`))
// 	}))
// 	defer srv.Close()
// 	p := Plugin{}
// 	result := p.Call(plugininterface.Args{
// 		Kind: "data",
// 		Name: "opencti",
// 		Config: cty.ObjectVal(map[string]cty.Value{
// 			"graphql_url": cty.StringVal(srv.URL),
// 			"auth_token":  cty.StringVal("token-123"),
// 		}),
// 		Args: cty.ObjectVal(map[string]cty.Value{
// 			"graphql_query": cty.StringVal("query issue { stixCoreRelationships { edges { node { x_opencti_stix_ids } } } }"),
// 		}),
// 	})
// 	s.Equal(want, result)
// }

// func (s *OpenCTIDataSourceTestSuite) TestNullURL() {
// 	p := Plugin{}
// 	result := p.Call(plugininterface.Args{
// 		Kind: "data",
// 		Name: "opencti",
// 		Config: cty.ObjectVal(map[string]cty.Value{
// 			"graphql_url": cty.NullVal(cty.String),
// 			"auth_token":  cty.NullVal(cty.String),
// 		}),
// 		Args: cty.ObjectVal(map[string]cty.Value{
// 			"graphql_query": cty.StringVal("query{user{id,name}}"),
// 		}),
// 	})
// 	s.Len(result.Diags, 1)
// 	s.Equal("Failed to parse config", result.Diags[0].Summary)
// }

// func (s *OpenCTIDataSourceTestSuite) TestEmptyURL() {
// 	p := Plugin{}
// 	result := p.Call(plugininterface.Args{
// 		Kind: "data",
// 		Name: "opencti",
// 		Config: cty.ObjectVal(map[string]cty.Value{
// 			"graphql_url": cty.StringVal(""),
// 			"auth_token":  cty.NullVal(cty.String),
// 		}),
// 		Args: cty.ObjectVal(map[string]cty.Value{
// 			"graphql_query": cty.StringVal("query{user{id,name}}"),
// 		}),
// 	})
// 	s.Len(result.Diags, 1)
// 	s.Equal("Failed to parse config", result.Diags[0].Summary)
// }

// func (s *OpenCTIDataSourceTestSuite) TestEmptyQuery() {
// 	p := Plugin{}
// 	result := p.Call(plugininterface.Args{
// 		Kind: "data",
// 		Name: "opencti",
// 		Config: cty.ObjectVal(map[string]cty.Value{
// 			"graphql_url": cty.StringVal("http://localhost"),
// 			"auth_token":  cty.NullVal(cty.String),
// 		}),
// 		Args: cty.ObjectVal(map[string]cty.Value{
// 			"graphql_query": cty.StringVal(""),
// 		}),
// 	})
// 	s.Len(result.Diags, 1)
// 	s.Equal("Failed to parse arguments", result.Diags[0].Summary)
// }

// func (s *OpenCTIDataSourceTestSuite) TestNullQuery() {
// 	p := Plugin{}
// 	result := p.Call(plugininterface.Args{
// 		Kind: "data",
// 		Name: "opencti",
// 		Config: cty.ObjectVal(map[string]cty.Value{
// 			"graphql_url": cty.StringVal("http://localhost"),
// 			"auth_token":  cty.NullVal(cty.String),
// 		}),
// 		Args: cty.ObjectVal(map[string]cty.Value{
// 			"graphql_query": cty.NullVal(cty.String),
// 		}),
// 	})
// 	s.Len(result.Diags, 1)
// 	s.Equal("Failed to parse arguments", result.Diags[0].Summary)
// }

// func jsonAny(s string) any {
// 	var v any
// 	err := json.Unmarshal([]byte(s), &v)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return v
// }
