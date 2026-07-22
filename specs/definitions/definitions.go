// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package definitions

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

const (
	BlockKindDocument = "document"
	BlockKindConfig   = "config"
	BlockKindContent  = "content"
	BlockKindPublish  = "publish"
	BlockKindData     = "data"
	BlockKindMeta     = "meta"
	BlockKindInput    = "input"
	BlockKindVars     = "vars"
	BlockKindSection  = "section"

	BlockKindDynamic = "dynamic"
	BlockKindFormat  = "format"

	BlockKindGlobalConfig    = "blackstork"
	BlockKindGlobalConfigOld = "fabric"

	BlockTypeRef = "ref"

	AttrRefBase      = "base"
	AttrTitle        = "title"
	AttrDependsOn    = "depends_on"
	AttrLocalVar     = "local_var"
	AttrRequiredVars = "required_vars"
	AttrIsIncluded   = "is_included"
	AttrDynamicItems = "items"
)

type BlockDef interface {
	GetHCLBlock() *hclsyntax.Block
	CtyType() cty.Type
	Kind() string
}

func ToCtyValue(b BlockDef) cty.Value {
	return cty.CapsuleVal(b.CtyType(), b)
}

// Key identifies a defined block
type Key struct {
	Kind string
	// Data source, content provider, formatter or publisher
	Runner string
	Name   string
}

func (k Key) FullName() string {
	name := k.Kind

	if k.Runner != "" {
		name += "." + k.Runner
	}

	if k.Name != "" {
		name += "." + k.Name
	}

	return name
}

func isValidKind(val string) bool {
	return slices.Contains([]string{
		BlockKindDocument,
		BlockKindConfig,
		BlockKindContent,
		BlockKindPublish,
		BlockKindData,
		BlockKindMeta,
		BlockKindVars,
		BlockKindSection,
		BlockKindGlobalConfig,
		BlockKindDynamic,
		BlockKindFormat,
	}, val)
}

func KeyFromName(val string) (*Key, error) {
	parts := strings.SplitN(val, ".", 3)
	var kind string
	var runner string
	var name string
	if len(parts) == 2 {
		if parts[0] != BlockKindSection {
			return nil, fmt.Errorf("invalid block type found in block name `%s`", val)
		}
		kind = BlockKindSection
		name = parts[1]
	} else if len(parts) == 3 {
		kind = parts[0]
		runner = parts[1]
		name = parts[2]
	} else {
		return nil, fmt.Errorf("error parsing block name `%s`", val)
	}

	if !isValidKind(kind) {
		return nil, fmt.Errorf("invalid block type found in a block name `%s`", val)
	}

	return &Key{
		Kind:   kind,
		Runner: runner,
		Name:   name,
	}, nil
}
