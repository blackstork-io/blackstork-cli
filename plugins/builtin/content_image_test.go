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
	"github.com/blackstork-io/blackstork-cli/plugin/plugintest"
)

type ImageGeneratorTestSuite struct {
	suite.Suite
	schema *plugin.ContentProvider
}

func TestImageGeneratorTestSuite(t *testing.T) {
	suite.Run(t, &ImageGeneratorTestSuite{})
}

func (s *ImageGeneratorTestSuite) SetupSuite() {
	s.schema = makeImageContentProvider()
}

func (s *ImageGeneratorTestSuite) TestSchema() {
	provider := makeImageContentProvider()
	s.Nil(provider.Config)
	s.NotNil(provider.Args)
	s.NotNil(provider.ContentFunc)
}

func (s *ImageGeneratorTestSuite) TestMissingImageSource() {
	plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		src = null
		alt = null
		`,
		nil, diagtest.Asserts{{
			diagtest.IsError,
			diagtest.SummaryContains("Attribute must be non-null"),
		}})
}

func (s *ImageGeneratorTestSuite) TestCallImageSourceEmpty() {
	plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		src = ""
		alt = null
		`,
		nil, diagtest.Asserts{{
			diagtest.IsError,
			diagtest.DetailContains(`"src"`, `can't be empty`),
		}})
}

func (s *ImageGeneratorTestSuite) TestCallImageSourceValid() {
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		src = "https://example.com/image.png"
		`,
		nil, nil)

	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,
	})
	s.Equal("![](https://example.com/image.png)", result.Content)
	s.Empty(diags)
}

func (s *ImageGeneratorTestSuite) TestCallImageSourceValidWithAlt() {
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		src = "https://example.com/image.png"
		alt = "alt text"
		`,
		nil, nil)

	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,
	})
	s.Equal("![alt text](https://example.com/image.png)", result.Content)
	s.Empty(diags)
}

func (s *ImageGeneratorTestSuite) TestCallImageSourceTemplateRender() {
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		src = "./{{ add 1 2 }}.png"
		`,
		nil, nil)

	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,
	})
	s.Equal("![](./3.png)", result.Content)
	s.Empty(diags)
}

func (s *ImageGeneratorTestSuite) TestCallImageAltTemplateRender() {
	args := plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		src = "./{{ add 1 2 }}.png"
		alt = "{{ add 2 3 }} alt text"
		`,
		nil, nil)

	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: args,
	})
	s.Equal("![5 alt text](./3.png)", result.Content)
	s.Empty(diags)
}
