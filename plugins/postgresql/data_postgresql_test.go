// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package postgresql

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

// IntegrationTestSuite is a test suite to test integration with real postgres instance
type IntegrationTestSuite struct {
	suite.Suite
	container *postgres.PostgresContainer
	connURL   string
	plugin    *plugin.Schema
	ctx       context.Context
}

func TestIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests")
	}
	suite.Run(t, &IntegrationTestSuite{})
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.plugin = Plugin("1.2.3")
	s.ctx = context.Background()
	container, err := postgres.Run(
		s.ctx, "docker.io/postgres:15.2-alpine",
		postgres.WithInitScripts(filepath.Join("testdata", "data.sql")),
		postgres.WithDatabase("testusr123"),
		postgres.WithPassword("testpsw123"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		if strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
			s.T().Skip("Docker not available for integration tests")
		} else {
			s.Require().NoError(err, "failed to start postgres container")
		}
	}
	s.container = container
	connURL, err := container.ConnectionString(s.ctx, "sslmode=disable")
	s.Require().NoError(err, "failed to get postgres connection string")
	s.connURL = connURL
	db, err := sql.Open("postgres", connURL)
	s.Require().NoError(err, "failed to open postgres database")
	err = db.Ping()
	s.Require().NoError(err, "failed to ping postgres database")
	err = db.Close()
	s.Require().NoError(err, "failed to close postgres database")
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.Require().NoError(s.container.Terminate(s.ctx), "failed to stop postgres container")
}

func (s *IntegrationTestSuite) TestSchema() {
	source := s.plugin.DataSources["postgresql"]
	s.Require().NotNil(source, "expected postgresql data source")
	s.NotNil(source.Config)
	s.NotNil(source.Args)
	s.NotNil(source.DataFunc)
}

func (s *IntegrationTestSuite) TestEmptySQLQuery() {
	data, diags := s.plugin.RetrieveData(s.ctx, "postgresql", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"database_url": cty.StringVal(s.connURL),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"sql_query": cty.StringVal(""),
			"sql_args":  cty.ListValEmpty(cty.DynamicPseudoType),
		}),
	})
	s.Nil(data)

	s.Equal(diags, diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Invalid arguments",
		Detail:   "sql_query is required",
	}})
}

func (s *IntegrationTestSuite) TestNilSQLQuery() {
	data, diags := s.plugin.RetrieveData(s.ctx, "postgresql", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"database_url": cty.StringVal(s.connURL),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"sql_query": cty.NullVal(cty.String),
			"sql_args":  cty.ListValEmpty(cty.DynamicPseudoType),
		}),
	})
	s.Nil(data)

	s.Equal(diags, diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Invalid arguments",
		Detail:   "sql_query is required",
	}})
}

func (s *IntegrationTestSuite) TestEmptyTable() {
	data, diags := s.plugin.RetrieveData(s.ctx, "postgresql", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"database_url": cty.StringVal(s.connURL),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"sql_query": cty.StringVal("SELECT * FROM testdata_empty"),
			"sql_args":  cty.ListValEmpty(cty.DynamicPseudoType),
		}),
	})
	s.Nil(diags)
	s.Equal(data, plugindata.List{})
}

func (s *IntegrationTestSuite) TestSelect() {
	data, diags := s.plugin.RetrieveData(s.ctx, "postgresql", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"database_url": cty.StringVal(s.connURL),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"sql_query": cty.StringVal("SELECT * FROM testdata"),
			"sql_args":  cty.ListValEmpty(cty.DynamicPseudoType),
		}),
	})
	s.Nil(diags)
	s.Equal(data, plugindata.List{
		plugindata.Map{
			"id":       plugindata.Number(1),
			"text_val": plugindata.String("text_1"),
			"int_val":  plugindata.Number(1),
			"bool_val": plugindata.Bool(true),
			"null_val": nil,
		},
		plugindata.Map{
			"id":       plugindata.Number(2),
			"text_val": plugindata.String("text_2"),
			"int_val":  plugindata.Number(2),
			"bool_val": plugindata.Bool(false),
			"null_val": plugindata.String("null_val_2"),
		},
	})
}

func (s *IntegrationTestSuite) TestSelectSomeFields() {
	data, diags := s.plugin.RetrieveData(s.ctx, "postgresql", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"database_url": cty.StringVal(s.connURL),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"sql_query": cty.StringVal("SELECT id, text_val AS text FROM testdata"),
			"sql_args":  cty.ListValEmpty(cty.DynamicPseudoType),
		}),
	})
	s.Nil(diags)
	s.Equal(data, plugindata.List{
		plugindata.Map{
			"id":   plugindata.Number(1),
			"text": plugindata.String("text_1"),
		},
		plugindata.Map{
			"id":   plugindata.Number(2),
			"text": plugindata.String("text_2"),
		},
	})
}

func (s *IntegrationTestSuite) TestSelectWithArgs() {
	data, diags := s.plugin.RetrieveData(s.ctx, "postgresql", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"database_url": cty.StringVal(s.connURL),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"sql_query": cty.StringVal("SELECT * FROM testdata WHERE text_val = $1 AND int_val = $2 AND bool_val = $3;"),
			"sql_args": cty.TupleVal([]cty.Value{
				cty.StringVal("text_2"),
				cty.NumberIntVal(2),
				cty.BoolVal(false),
			}),
		}),
	})
	s.Nil(diags)
	s.Equal(plugindata.List{
		plugindata.Map{
			"id":       plugindata.Number(2),
			"text_val": plugindata.String("text_2"),
			"int_val":  plugindata.Number(2),
			"bool_val": plugindata.Bool(false),
			"null_val": plugindata.String("null_val_2"),
		},
	}, data)
}

func (s *IntegrationTestSuite) TestSelectWithMissingArgs() {
	data, diags := s.plugin.RetrieveData(s.ctx, "postgresql", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"database_url": cty.StringVal(s.connURL),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"sql_query": cty.StringVal("SELECT * FROM testdata WHERE bool_val = $1;"),
			"sql_args":  cty.NilVal,
		}),
	})
	s.Nil(data)
	s.Equal(diags, diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to query database",
		Detail:   "pq: there is no parameter $1",
	}})
}

func (s *IntegrationTestSuite) TestCancellation() {
	ctx, cancel := context.WithCancel(s.ctx)
	cancel()
	data, diags := s.plugin.RetrieveData(ctx, "postgresql", &plugin.RetrieveDataParams{
		Config: dataspec.NewBlock([]string{"config"}, map[string]cty.Value{
			"database_url": cty.StringVal(s.connURL),
		}),
		Args: dataspec.NewBlock([]string{"args"}, map[string]cty.Value{
			"sql_query": cty.StringVal("SELECT * FROM testdata"),
			"sql_args":  cty.ListValEmpty(cty.DynamicPseudoType),
		}),
	})
	s.Nil(data)
	s.Equal(diags, diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to query database",
		Detail:   "context canceled",
	}})
}
