// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package builtin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugin/plugintest"
)

type ListGeneratorTestSuite struct {
	suite.Suite
	schema *plugin.ContentProvider
}

func TestListGeneratorTestSuite(t *testing.T) {
	suite.Run(t, &ListGeneratorTestSuite{})
}

func (s *ListGeneratorTestSuite) SetupSuite() {
	s.schema = makeListContentProvider()
}

func (s *ListGeneratorTestSuite) TestSchema() {
	s.NotNil(s.schema)
	s.Nil(s.schema.Config)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.ContentFunc)
}

func (s *ListGeneratorTestSuite) TestNilQueryResult() {
	dataCtx := plugindata.Map{}
	plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		items = null
	`, dataCtx, diagtest.Asserts{{
		diagtest.IsError,
		diagtest.SummaryContains("Attribute", "non-null"),
	}})
}

func (s *ListGeneratorTestSuite) TestNonArrayQueryResult() {
	dataCtx := plugindata.Map{}
	plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		items = "not_an_array"
	`, dataCtx, diagtest.Asserts{{
		diagtest.IsError,
		diagtest.SummaryContains("Incorrect", "type"),
		diagtest.DetailContains("list of jq queriable required"),
	}})
}

func (s *ListGeneratorTestSuite) TestUnordered() {
	dataCtx := plugindata.Map{}
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		items = ["bar", "baz"]
		item_template = "foo {{.}}"
		format = "unordered"
	`, dataCtx, diagtest.Asserts{})

	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args:        args,
		DataContext: dataCtx,
	})
	// s.Equal("* foo bar\n* foo baz\n", mdprint.PrintString(result.Content))
	s.Equal("* foo bar\n* foo baz\n", result.Content)
	s.Empty(diags)
}

func (s *ListGeneratorTestSuite) TestOrdered() {
	dataCtx := plugindata.Map{}
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		items = ["bar", "baz"]
		item_template = "foo {{.}}"
		format = "ordered"
	`, dataCtx, diagtest.Asserts{})

	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args:        args,
		DataContext: dataCtx,
	})
	// s.Equal("1. foo bar\n2. foo baz\n", mdprint.PrintString(result.Content))
	s.Equal("1. foo bar\n2. foo baz\n", result.Content)
	s.Empty(diags)
}

func (s *ListGeneratorTestSuite) TestTaskList() {
	dataCtx := plugindata.Map{}
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		items = ["bar", "baz"]
		item_template = "foo {{.}}"
		format = "tasklist"
	`, dataCtx, diagtest.Asserts{})

	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args:        args,
		DataContext: dataCtx,
	})
	// s.Equal("* [ ] foo bar\n* [ ] foo baz\n", mdprint.PrintString(result.Content))
	s.Equal("* [ ] foo bar\n* [ ] foo baz\n", result.Content)
	s.Empty(diags)
}

func (s *ListGeneratorTestSuite) TestBasic() {
	dataCtx := plugindata.Map{}
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		items = ["bar", "baz"]
		item_template = "foo {{.}}"
	`, dataCtx, diagtest.Asserts{})

	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args:        args,
		DataContext: dataCtx,
	})
	// s.Equal("* foo bar\n* foo baz\n", mdprint.PrintString(result.Content))
	s.Equal("* foo bar\n* foo baz\n", result.Content)
	s.Empty(diags)
}

func (s *ListGeneratorTestSuite) TestAdvanced() {
	dataCtx := plugindata.Map{}
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		items = [
			{
				bar = "bar1",
				baz = "baz1",
			},
			{
				bar = "bar2",
				baz = "baz2",
			}
		]
		item_template = "foo {{.bar}} {{.baz | upper}}"
	`, dataCtx, diagtest.Asserts{})

	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args:        args,
		DataContext: dataCtx,
	})
	// s.Equal("* foo bar1 BAZ1\n* foo bar2 BAZ2\n", mdprint.PrintString(result.Content))
	s.Equal("* foo bar1 BAZ1\n* foo bar2 BAZ2\n", result.Content)
	s.Empty(diags)
}

func (s *ListGeneratorTestSuite) TestEmptyQueryResult() {
	dataCtx := plugindata.Map{}
	plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		items = []
		item_template = "foo {{.}}"
	`, dataCtx, diagtest.Asserts{{
		diagtest.IsError,
		diagtest.DetailContains("items", "can't be empty"),
	}})
}

func (s *ListGeneratorTestSuite) TestMissingItemTemplate() {
	dataCtx := plugindata.Map{}
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		items = ["bar", "baz"]
	`, dataCtx, diagtest.Asserts{})

	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args:        args,
		DataContext: dataCtx,
	})
	// s.Equal("* bar\n* baz\n", mdprint.PrintString(result.Content))
	s.Equal("* bar\n* baz\n", result.Content)
	s.Empty(diags)
}
