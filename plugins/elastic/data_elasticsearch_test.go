// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"

	es "github.com/elastic/go-elasticsearch/v8"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go/modules/elasticsearch"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugin/plugintest"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

const (
	testIndex = "test_index"
)

// IntegrationTestSuite is a test suite to test integration with real elasticsearch
type IntegrationTestSuite struct {
	suite.Suite
	container *elasticsearch.ElasticsearchContainer
	client    *es.Client
	schema    *plugin.DataSource
	cfg       *dataspec.Block
	ctx       context.Context
}

func TestIntegrationSuite(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	if testing.Short() {
		t.Skip("skipping integration tests")
	}
	suite.Run(t, &IntegrationTestSuite{})
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.ctx = context.Background()
	container, err := elasticsearch.Run(
		s.ctx, "docker.io/elasticsearch:8.9.0",
		elasticsearch.WithPassword("password123"),
	)
	if err != nil {
		if strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
			s.T().Skip("Docker not available for integration tests")
		} else {
			s.Require().NoError(err, "failed to start elasticsearch container")
		}
	}

	s.container = container
	client, err := es.NewClient(es.Config{
		Addresses: []string{
			container.Settings.Address,
		},
		Username: "elastic",
		Password: container.Settings.Password,
		CACert:   container.Settings.CACert,
	})
	s.Require().NoError(err, "failed to create elasticsearch client")
	s.client = client

	s.schema = makeElasticSearchDataSource()

	s.cfg = plugintest.NewTestDecoder(s.T(), s.schema.Config).
		SetAttr("base_url", cty.StringVal(s.container.Settings.Address)).
		SetAttr("basic_auth_username", cty.StringVal("elastic")).
		SetAttr("basic_auth_password", cty.StringVal("password123")).
		SetAttr("ca_certs", cty.StringVal(string(s.container.Settings.CACert))).
		Decode()
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.Require().NoError(s.container.Terminate(s.ctx), "failed to stop elasticsearch container")
}

type testDataObject struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Active bool   `json:"active"`
	Age    int    `json:"age"`
	Name   string `json:"name"`
}

func (s *IntegrationTestSuite) SetupTest() {
	file, err := os.ReadFile("testdata/data.json")
	s.Require().NoError(err, "failed to read data.json")
	dataList := []testDataObject{}
	err = json.Unmarshal(file, &dataList)
	s.Require().NoError(err, "failed to unmarshal data.json")
	res, err := s.client.Indices.Create(testIndex)
	s.Require().NoError(err, fmt.Sprintf("failed to create index %s", testIndex))
	s.Require().Equal(http.StatusOK, res.StatusCode)
	for _, data := range dataList {
		dataBytes, err := json.Marshal(data)
		s.Require().NoError(err, "failed to marshal data")
		res, err := s.client.Create(testIndex, data.ID, bytes.NewReader(dataBytes))
		s.Require().NoError(err, "failed to index data")
		s.Require().False(res.IsError(), "failed to index data")
		res, err = s.client.Index(testIndex, bytes.NewReader(dataBytes), s.client.Index.WithDocumentID(data.ID))
		s.Require().NoError(err, "failed to index data")
		s.Require().False(res.IsError(), "failed to index data")
	}
	res, err = s.client.Indices.Refresh()
	s.Require().NoError(err, "failed to refresh indices")
	s.Require().False(res.IsError(), "failed to refresh indices")
}

func (s *IntegrationTestSuite) TearDownTest() {
	res, err := s.client.Indices.Delete([]string{testIndex})
	s.Require().NoError(err, "failed to delete indices")
	s.Require().False(res.IsError(), "failed to delete indices: %s", res.String())
}

func (s *IntegrationTestSuite) TestSearchDefaults() {
	args := plugintest.NewTestDecoder(s.T(), s.schema.Args).
		SetAttr("index", cty.StringVal(testIndex)).
		Decode()
	data, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: s.cfg,
		Args:   args,
	})
	s.Require().Nil(diags)
	m := data.(plugindata.List)
	raw, err := json.MarshalIndent(m, "", "  ")
	s.Require().NoError(err, "failed to marshal data: %v", err)
	s.JSONEq(`[
		  {
			"_id": "54f7a815-eac5-4f7c-a339-5fefd0f54967",
			"_index": "test_index",
			"_score": 1,
			"_source": {
			  "active": false,
			  "age": 39,
			  "id": "54f7a815-eac5-4f7c-a339-5fefd0f54967",
			  "name": "Davidson",
			  "type": "foo"
			}
		  },
		  {
			"_id": "0c68e63d-daaa-4a62-92e6-e855bd144fb6",
			"_index": "test_index",
			"_score": 1,
			"_source": {
			  "active": false,
			  "age": 20,
			  "id": "0c68e63d-daaa-4a62-92e6-e855bd144fb6",
			  "name": "Thompson",
			  "type": "bar"
			}
		  },
		  {
			"_id": "a117a5e6-23d0-4daa-be3c-a70900ca4163",
			"_index": "test_index",
			"_score": 1,
			"_source": {
			  "active": true,
			  "age": 21,
			  "id": "a117a5e6-23d0-4daa-be3c-a70900ca4163",
			  "name": "Armstrong",
			  "type": "foo"
			}
		  }
		]`, string(raw))
}

func (s *IntegrationTestSuite) TestSearchFields() {
	args := plugintest.NewTestDecoder(s.T(), s.schema.Args).
		SetAttr("index", cty.StringVal(testIndex)).
		SetAttr("only_hits", cty.BoolVal(false)).
		SetAttr("fields", cty.ListVal([]cty.Value{cty.StringVal("name"), cty.StringVal("age")})).
		Decode()
	data, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: s.cfg,
		Args:   args,
	})
	s.Require().Nil(diags)
	m := data.(plugindata.Map)
	raw, err := json.MarshalIndent(m["hits"], "", "  ")
	s.Require().NoError(err, "failed to marshal data: %v", err)
	s.JSONEq(`{
		"hits": [
		  {
			"_id": "54f7a815-eac5-4f7c-a339-5fefd0f54967",
			"_index": "test_index",
			"_score": 1,
			"_source": {
			  "age": 39,
			  "name": "Davidson"
			}
		  },
		  {
			"_id": "0c68e63d-daaa-4a62-92e6-e855bd144fb6",
			"_index": "test_index",
			"_score": 1,
			"_source": {
			  "age": 20,
			  "name": "Thompson"
			}
		  },
		  {
			"_id": "a117a5e6-23d0-4daa-be3c-a70900ca4163",
			"_index": "test_index",
			"_score": 1,
			"_source": {
			  "age": 21,
			  "name": "Armstrong"
			}
		  }
		],
		"max_score": 1,
		"total": {
		  "relation": "eq",
		  "value": 3
		}
	  }`, string(raw))
}

func (s *IntegrationTestSuite) TestSearchQueryString() {
	args := plugintest.NewTestDecoder(s.T(), s.schema.Args).
		SetAttr("index", cty.StringVal(testIndex)).
		SetAttr("query_string", cty.StringVal("type:foo")).
		SetAttr("only_hits", cty.BoolVal(false)).
		Decode()
	data, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: s.cfg,
		Args:   args,
	})
	s.Require().Nil(diags)
	m := data.(plugindata.Map)
	raw, err := json.MarshalIndent(m["hits"], "", "  ")
	s.Require().NoError(err, "failed to marshal data: %v", err)
	s.JSONEq(`{
		"hits": [
		  {
			"_id": "54f7a815-eac5-4f7c-a339-5fefd0f54967",
			"_index": "test_index",
			"_score": 0.44183272,
			"_source": {
			  "active": false,
			  "age": 39,
			  "id": "54f7a815-eac5-4f7c-a339-5fefd0f54967",
			  "name": "Davidson",
			  "type": "foo"
			}
		  },
		  {
			"_id": "a117a5e6-23d0-4daa-be3c-a70900ca4163",
			"_index": "test_index",
			"_score": 0.44183272,
			"_source": {
			  "active": true,
			  "age": 21,
			  "id": "a117a5e6-23d0-4daa-be3c-a70900ca4163",
			  "name": "Armstrong",
			  "type": "foo"
			}
		  }
		],
		"max_score": 0.44183272,
		"total": {
		  "relation": "eq",
		  "value": 2
		}
	  }`, string(raw))
}

func (s *IntegrationTestSuite) TestSearchQuery() {
	args := plugintest.NewTestDecoder(s.T(), s.schema.Args).
		SetAttr("index", cty.StringVal(testIndex)).
		SetAttr("query", cty.MapVal(map[string]cty.Value{
			"match_all": cty.MapValEmpty(cty.DynamicPseudoType),
		})).
		SetAttr("only_hits", cty.BoolVal(false)).
		Decode()
	data, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: s.cfg,
		Args:   args,
	})
	s.Require().Nil(diags)
	m := data.(plugindata.Map)
	raw, err := json.MarshalIndent(m["hits"], "", "  ")
	s.Require().NoError(err, "failed to marshal data: %v", err)
	s.JSONEq(`{
		"hits": [
			{
				"_id": "54f7a815-eac5-4f7c-a339-5fefd0f54967",
				"_index": "test_index",
				"_score": 1,
				"_source": {
					"active": false,
					"age": 39,
					"id": "54f7a815-eac5-4f7c-a339-5fefd0f54967",
					"name": "Davidson",
					"type": "foo"
				}
			},
			{
				"_id": "0c68e63d-daaa-4a62-92e6-e855bd144fb6",
				"_index": "test_index",
				"_score": 1,
				"_source": {
					"active": false,
					"age": 20,
					"id": "0c68e63d-daaa-4a62-92e6-e855bd144fb6",
					"name": "Thompson",
					"type": "bar"
				}
			},
			{
				"_id": "a117a5e6-23d0-4daa-be3c-a70900ca4163",
				"_index": "test_index",
				"_score": 1,
				"_source": {
					"active": true,
					"age": 21,
					"id": "a117a5e6-23d0-4daa-be3c-a70900ca4163",
					"name": "Armstrong",
					"type": "foo"
				}
			}
		],
		"max_score": 1,
		"total": {
			"relation": "eq",
			"value": 3
		}
	}`, string(raw))
}

func (s *IntegrationTestSuite) TestSearchSize() {
	args := plugintest.NewTestDecoder(s.T(), s.schema.Args).
		SetAttr("index", cty.StringVal(testIndex)).
		SetAttr("only_hits", cty.BoolVal(false)).
		SetAttr("size", cty.NumberIntVal(1)).
		Decode()
	data, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: s.cfg,
		Args:   args,
	})
	s.Require().Nil(diags)
	m := data.(plugindata.Map)
	raw, err := json.MarshalIndent(m["hits"], "", "  ")
	s.Require().NoError(err, "failed to marshal data: %v", err)
	s.JSONEq(`{
		"hits": [
		  {
			"_id": "54f7a815-eac5-4f7c-a339-5fefd0f54967",
			"_index": "test_index",
			"_score": 1,
			"_source": {
			  "active": false,
			  "age": 39,
			  "id": "54f7a815-eac5-4f7c-a339-5fefd0f54967",
			  "name": "Davidson",
			  "type": "foo"
			}
		  }
		],
		"max_score": 1,
		"total": {
		  "relation": "eq",
		  "value": 3
		}
	  }`, string(raw))
}

func (s *IntegrationTestSuite) TestGetByID() {
	args := plugintest.NewTestDecoder(s.T(), s.schema.Args).
		SetAttr("id", cty.StringVal("0c68e63d-daaa-4a62-92e6-e855bd144fb6")).
		SetAttr("index", cty.StringVal(testIndex)).
		SetAttr("only_hits", cty.BoolVal(false)).
		Decode()
	data, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: s.cfg,
		Args:   args,
	})
	s.Require().Nil(diags)
	m := data.(plugindata.Map)
	raw, err := json.MarshalIndent(m, "", "  ")
	s.Require().NoError(err, "failed to marshal data: %v", err)
	s.JSONEq(`{
		"_id": "0c68e63d-daaa-4a62-92e6-e855bd144fb6",
		"_index": "test_index",
		"_primary_term": 1,
		"_seq_no": 3,
		"_source": {
		  "active": false,
		  "age": 20,
		  "id": "0c68e63d-daaa-4a62-92e6-e855bd144fb6",
		  "name": "Thompson",
		  "type": "bar"
		},
		"_version": 2,
		"found": true
	}`, string(raw))
}

func (s *IntegrationTestSuite) TestGetByIDFields() {
	args := plugintest.NewTestDecoder(s.T(), s.schema.Args).
		SetAttr("id", cty.StringVal("0c68e63d-daaa-4a62-92e6-e855bd144fb6")).
		SetAttr("index", cty.StringVal(testIndex)).
		SetAttr("only_hits", cty.BoolVal(false)).
		SetAttr("fields", cty.ListVal([]cty.Value{cty.StringVal("name"), cty.StringVal("age")})).
		Decode()
	data, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: s.cfg,
		Args:   args,
	})
	s.Require().Nil(diags)
	m := data.(plugindata.Map)
	raw, err := json.MarshalIndent(m, "", "  ")
	s.Require().NoError(err, "failed to marshal data: %v", err)
	s.JSONEq(`{
		"_id": "0c68e63d-daaa-4a62-92e6-e855bd144fb6",
		"_index": "test_index",
		"_primary_term": 1,
		"_seq_no": 3,
		"_source": {
		  "age": 20,
		  "name": "Thompson"
		},
		"_version": 2,
		"found": true
	}`, string(raw))
}

func (s *IntegrationTestSuite) TestGetByIDNotFound() {
	args := plugintest.NewTestDecoder(s.T(), s.schema.Args).
		SetAttr("id", cty.StringVal("00000000-0000-0000-0000-000000000000")).
		SetAttr("index", cty.StringVal(testIndex)).
		SetAttr("only_hits", cty.BoolVal(false)).
		Decode()
	data, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: s.cfg,
		Args:   args,
	})
	s.Nil(data)
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to fetch data",
		Detail:   "failed to get document: [404 Not Found] {\"_index\":\"test_index\",\"_id\":\"00000000-0000-0000-0000-000000000000\",\"found\":false}",
	}}, diags)
}

func (s *IntegrationTestSuite) TestScrollSearch() {
	args := plugintest.NewTestDecoder(s.T(), s.schema.Args).
		SetAttr("index", cty.StringVal(testIndex)).
		SetAttr("only_hits", cty.BoolVal(false)).
		SetAttr("size", cty.NumberIntVal(10000+1)). // to force scroll search
		Decode()
	data, diags := s.schema.DataFunc(s.ctx, &plugin.RetrieveDataParams{
		Config: s.cfg,
		Args:   args,
	})
	s.Require().Nil(diags, fmt.Sprintf("Received diagnostics: %s", diags))
	dataMap := data.(plugindata.Map)
	hitsEnvelope := dataMap["hits"].(plugindata.Map)
	hits := hitsEnvelope["hits"].(plugindata.List)

	s.Equal(3, len(hits), fmt.Sprintf("Hits received: %s", hits))
}

func (s *IntegrationTestSuite) TestScrollSearchSteps() {
	args := plugintest.NewTestDecoder(s.T(), s.schema.Args).
		SetAttr("index", cty.StringVal(testIndex)).
		SetAttr("only_hits", cty.BoolVal(false)).
		SetAttr("size", cty.NumberIntVal(5)).
		Decode()
	// There are only 3 results, so with the size 5 and step size 1,
	// we should hit 4 requests:
	// - initial search request (1 result)
	// - 3 scroll requests: 2 with 1 result and one with an empty result to break the loop
	data, err := searchWithScrollConfigurable(s.client, args, 5, 1)

	s.Require().Nil(err, fmt.Sprintf("Received diagnostics: %s", err))
	dataMap := data.(plugindata.Map)
	hitsEnvelope := dataMap["hits"].(plugindata.Map)
	hits := hitsEnvelope["hits"].(plugindata.List)

	s.Equal(3, len(hits), fmt.Sprintf("Hits received: %s", hits))

	raw, err := json.MarshalIndent(hitsEnvelope, "", "  ")
	s.Require().NoError(err, "failed to marshal data: %v", err)
	s.JSONEq(`{
		"hits": [
			{
				"_id": "54f7a815-eac5-4f7c-a339-5fefd0f54967",
				"_index": "test_index",
				"_score": 1,
				"_source": {
					"active": false,
					"age": 39,
					"id": "54f7a815-eac5-4f7c-a339-5fefd0f54967",
					"name": "Davidson",
					"type": "foo"
				}
			},
			{
				"_id": "0c68e63d-daaa-4a62-92e6-e855bd144fb6",
				"_index": "test_index",
				"_score": 1,
				"_source": {
					"active": false,
					"age": 20,
					"id": "0c68e63d-daaa-4a62-92e6-e855bd144fb6",
					"name": "Thompson",
					"type": "bar"
				}
			},
			{
				"_id": "a117a5e6-23d0-4daa-be3c-a70900ca4163",
				"_index": "test_index",
				"_score": 1,
				"_source": {
					"active": true,
					"age": 21,
					"id": "a117a5e6-23d0-4daa-be3c-a70900ca4163",
					"name": "Armstrong",
					"type": "foo"
				}
			}
		],
		"max_score": 1,
		"total": {
			"relation": "eq",
			"value": 3
		}
	}`, string(raw))
}
