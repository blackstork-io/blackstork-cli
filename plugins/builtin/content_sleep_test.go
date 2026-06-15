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
	"io"
	"log/slog"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugintest"
)

type SleepContentTestSuite struct {
	suite.Suite
	schema *plugin.ContentProvider
}

func TestSleepContentSuite(t *testing.T) {
	suite.Run(t, &SleepContentTestSuite{})
}

func (s *SleepContentTestSuite) SetupSuite() {
	s.schema = makeSleepContentProvider(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func (s *SleepContentTestSuite) TestSchema() {
	s.Nil(s.schema.Config)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.ContentFunc)
}

func (s *SleepContentTestSuite) TestMissingDuration() {
	plugintest.NewTestDecoder(s.T(), s.schema.Args).Decode()
}

func (s *SleepContentTestSuite) TestCustom() {
	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("duration", cty.StringVal("123ms")).
			Decode(),
	})
	s.Require().Empty(diags)
	s.Equal("Slept for 123ms.", result.Content)
}

func (s *SleepContentTestSuite) TestDefault() {
	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args:        plugintest.NewTestDecoder(s.T(), s.schema.Args).Decode(),
		DataContext: nil,
	})
	s.Require().Empty(diags)
	s.Equal("Slept for 1s.", result.Content)
}

func (s *SleepContentTestSuite) TestCallInvalidDuration() {
	ctx := context.Background()
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("duration", cty.StringVal("invalid")).
			Decode(),
	})
	s.Require().Nil(result)
	s.Require().Len(diags, 1)
	s.Equal(hcl.DiagError, diags[0].Severity)
	s.Equal("Invalid duration", diags[0].Summary)
}
