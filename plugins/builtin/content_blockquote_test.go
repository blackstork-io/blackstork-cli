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

type BlockQuoteTestSuite struct {
	suite.Suite
	schema *plugin.ContentProvider
}

func TestBlockQuoteSuite(t *testing.T) {
	suite.Run(t, &BlockQuoteTestSuite{})
}

func (s *BlockQuoteTestSuite) SetupSuite() {
	s.schema = makeBlockQuoteContentProvider()
}

func (s *BlockQuoteTestSuite) TestSchema() {
	s.Nil(s.schema.Config)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.ContentFunc)
}

func (s *BlockQuoteTestSuite) TestMissingText() {
	plugintest.DecodeAndAssert(s.T(), s.schema.Args, ``, nil, diagtest.Asserts{
		{
			diagtest.IsError,
			diagtest.DetailContains(`The attribute "value" is required`),
		},
	})
}

func (s *BlockQuoteTestSuite) TestNullText() {
	plugintest.DecodeAndAssert(s.T(), s.schema.Args, `value = null`, nil, diagtest.Asserts{
		{
			diagtest.IsError,
			diagtest.SummaryContains(`Attribute must be non-null`),
		},
	})
}

func (s *BlockQuoteTestSuite) TestCallBlockquote() {
	ctx := context.Background()
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		value = "Hello {{.name}}!"
	`, nil, nil)
	content, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,
		DataContext: plugindata.Map{
			"name": plugindata.String("World"),
		},
	})
	s.Empty(diags)
	s.Equal(
		plugindata.String("> Hello World!"),
		content.Content.AsData()["markdown"],
	)
}

func (s *BlockQuoteTestSuite) TestCallBlockquoteMultiline() {
	ctx := context.Background()
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		value = "Hello\n{{.name}}\nfor you!"
	`, nil, nil)
	content, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,
		DataContext: plugindata.Map{
			"name": plugindata.String("World"),
		},
	})
	s.Empty(diags)
	s.Equal(
		plugindata.String("> Hello\n> World\n> for you!"),
		content.Content.AsData()["markdown"],
	)
}

func (s *BlockQuoteTestSuite) TestCallBlockquoteMultilineDoubleNewline() {
	ctx := context.Background()
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		value = "Hello\n{{.name}}\n\nfor you!"
	`, nil, nil)
	content, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,
		DataContext: plugindata.Map{
			"name": plugindata.String("World"),
		},
	})
	s.Empty(diags)
	s.Equal(
		plugindata.String("> Hello\n> World\n> \n> for you!"),
		content.Content.AsData()["markdown"],
	)
}
