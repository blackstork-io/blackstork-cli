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

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

// EvalKey identifies a content tree block
type EvalKey struct {
	Kind   string `json:"kind"             validate:"required,min=1"`
	Runner string `json:"runner,omitempty"`
	Name   string `json:"name"             validate:"required,min=1"`
}

func (key EvalKey) AsName() string {
	name := key.Kind
	if key.Runner != "" {
		name += "." + key.Runner
	}
	name += "." + key.Name
	return name
}

func (key EvalKey) AsPath() []string {
	path := []string{key.Kind}
	if key.Runner != "" {
		path = append(path, key.Runner)
	}
	path = append(path, key.Name)
	return path
}

func EvalKeyFromDefKey(key definitions.Key) EvalKey {
	return EvalKey{
		Kind:   key.Kind,
		Runner: key.Runner,
		Name:   key.Name,
	}
}

type ContentTreeEvalBlock interface {
	ID() string
	Kind() string
	EvalKey() EvalKey

	Clone(suffix string) ContentTreeEvalBlock

	isContentTreeEvalBlock()
}

type RenderableContent interface {
	RenderContent(ctx context.Context, data plugindata.Map) (plugin.Content, diagnostics.Diag)
	EvalKey() EvalKey

	Kind() string
	ID() string

	Meta() plugindata.Map

	GetDataCtx() *plugindata.Map
	GetSrcRange() *hcl.Range
}

func LoadContent(
	ctx context.Context,
	providers ContentProviders,
	block definitions.ContentTreeBlock,
) (_ ContentTreeEvalBlock, diags diagnostics.Diag) {
	var treeBlock ContentTreeEvalBlock
	switch block := block.(type) {
	case *definitions.ContentBlock:
		treeBlock, diags = LoadPluginContentAction(ctx, providers, block)
	case *definitions.Section:
		treeBlock, diags = LoadSection(ctx, providers, block)
	case *definitions.Dynamic:
		treeBlock, diags = LoadDynamic(ctx, providers, block)
	default:
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Error while loading content: unsupported content tree block type: %T", block),
			Detail:   "Content tree block must be either `content`, `section` or `dynamic`",
		})
	}
	if diags.HasErrors() {
		return nil, diags
	}
	return treeBlock, diags
}
