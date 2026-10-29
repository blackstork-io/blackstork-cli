// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package eval

import (
	"context"
	"fmt"
	"iter"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/require"

	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

type publisherRegistryStub map[string]*plugin.Publisher

func (r publisherRegistryStub) Publisher(name string) (*plugin.Publisher, bool) {
	publisher, ok := r[name]
	return publisher, ok
}

func (r publisherRegistryStub) Publishers() iter.Seq2[string, *plugin.Publisher] {
	return func(yield func(string, *plugin.Publisher) bool) {
		for name, publisher := range r {
			if !yield(name, publisher) {
				return
			}
		}
	}
}

type formatterRegistryStub map[string]*plugin.Formatter

func (r formatterRegistryStub) Formatter(name string) (*plugin.Formatter, bool) {
	formatter, ok := r[name]
	return formatter, ok
}

func (r formatterRegistryStub) Formatters() iter.Seq2[string, *plugin.Formatter] {
	return func(yield func(string, *plugin.Formatter) bool) {
		for name, formatter := range r {
			if !yield(name, formatter) {
				return
			}
		}
	}
}

func TestLoadPluginPublishActionRequiresFormatReference(t *testing.T) {
	publishers := publisherRegistryStub{
		"test_publisher": {
			Args: &dataspec.RootSpec{},
		},
	}
	block := &definitions.PublishBlock{
		Source:     mustExecBlockDef(t, definitions.BlockKindPublish, "test_publisher"),
		RunnerName: "test_publisher",
	}

	action, diags := LoadPluginPublishAction(
		context.Background(),
		publishers,
		formatterRegistryStub{},
		block,
		plugindata.Map{},
	)

	require.Nil(t, action)
	require.True(t, diags.HasErrors())
	require.Contains(t, diags[0].Summary, "Missing formatter reference")
	require.Contains(t, diags[0].Detail, "format_ref")
}

func TestLoadPluginPublishActionRejectsUnsupportedFormat(t *testing.T) {
	publishers := publisherRegistryStub{
		"test_publisher": {
			Args:               &dataspec.RootSpec{},
			AcceptedFormatters: []string{"md"},
		},
	}
	formatters := formatterRegistryStub{
		"custom_html": {
			Format: "html",
			Args:   &dataspec.RootSpec{},
		},
	}
	block := publishBlockWithFormat(t, "test_publisher", "custom_html")

	action, diags := LoadPluginPublishAction(
		context.Background(),
		publishers,
		formatters,
		block,
		plugindata.Map{},
	)

	require.Nil(t, action)
	require.True(t, diags.HasErrors())
	require.Contains(t, diags[0].Summary, "Unsupported publisher format")
	require.Contains(t, diags[0].Detail, "html")
	require.Contains(t, diags[0].Detail, "md")
}

func TestLoadPluginPublishActionAcceptsFormatterOutputFormat(t *testing.T) {
	publisher := &plugin.Publisher{
		Args:               &dataspec.RootSpec{},
		AcceptedFormatters: []string{"html"},
	}
	formatter := &plugin.Formatter{
		Format: "html",
		Args:   &dataspec.RootSpec{},
	}
	block := publishBlockWithFormat(t, "test_publisher", "custom_html")

	action, diags := LoadPluginPublishAction(
		context.Background(),
		publisherRegistryStub{"test_publisher": publisher},
		formatterRegistryStub{"custom_html": formatter},
		block,
		plugindata.Map{},
	)

	require.False(t, diags.HasErrors())
	require.NotNil(t, action)
	require.Same(t, publisher, action.Publisher)
	require.Same(t, formatter, action.Format.Formatter)
}

func publishBlockWithFormat(t *testing.T, publisher, formatter string) *definitions.PublishBlock {
	t.Helper()
	return &definitions.PublishBlock{
		Source:     mustExecBlockDef(t, definitions.BlockKindPublish, publisher),
		RunnerName: publisher,
		Format: &definitions.FormatBlock{
			Source:     mustExecBlockDef(t, definitions.BlockKindFormat, formatter),
			RunnerName: formatter,
		},
	}
}

func mustExecBlockDef(t *testing.T, kind, runner string) *definitions.ExecBlockDef {
	t.Helper()
	source := fmt.Appendf(nil, "%s %q {}", kind, runner)
	file, diags := hclsyntax.ParseConfig(source, "test.blackstork.hcl", hcl.InitialPos)
	require.False(t, diags.HasErrors())
	body := file.Body.(*hclsyntax.Body)
	require.Len(t, body.Blocks, 1)
	return &definitions.ExecBlockDef{Block: body.Blocks[0]}
}
