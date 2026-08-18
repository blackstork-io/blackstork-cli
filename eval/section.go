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
	"reflect"

	"github.com/google/uuid"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/deferred"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

type Section struct {
	source *definitions.Section

	blockName string

	title    ContentTreeEvalBlock
	children []ContentTreeEvalBlock

	meta *definitions.MetaBlock
	vars *definitions.Vars

	requiredVars []string
	dependsOn    []EvalKey

	isIncluded *dataspec.Attr

	// unique identifier
	id string

	// To be filled in during unpacking and data propagation
	dataCtx *plugindata.Map

	titleToRender    RenderableContent
	childrenToRender []RenderableContent
}

func (s *Section) EvalKey() EvalKey {
	return EvalKey{
		Kind: definitions.BlockKindSection,
		Name: s.blockName, // can be overriden by ref base or dynamic block
	}
}

func (s *Section) Kind() string {
	return s.source.GetSourceKind()
}

func (s *Section) GetDataCtx() *plugindata.Map {
	return s.dataCtx
}

func (s *Section) Meta() plugindata.Map {
	if s.meta == nil {
		return plugindata.Map{}
	}
	return s.meta.AsPluginData()
}

func (s *Section) addNameSuffix(val string) {
	s.blockName += ":" + val
}

func (s *Section) ID() string {
	if s.id == "" {
		s.makeNewID()
	}
	return s.id
}

func (s *Section) makeNewID() {
	uid, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Sprintf("Error generating a UUID v7: %v", err))
	}
	s.id = uid.String()
}

func (s *Section) Clone(suffix string) ContentTreeEvalBlock {
	var titleClone ContentTreeEvalBlock

	if s.title != nil {
		titleClone = s.title.Clone(suffix)
	}

	res := &Section{
		source:    s.source,
		blockName: s.blockName,

		title: titleClone,
		children: utils.FnMap(
			s.children,
			func(b ContentTreeEvalBlock) ContentTreeEvalBlock { return b.Clone(suffix) },
		),

		meta: s.meta,
		vars: s.vars,

		requiredVars: s.requiredVars,
		dependsOn:    s.dependsOn,

		isIncluded: s.isIncluded,
	}
	res.makeNewID()

	if suffix != "" {
		res.addNameSuffix(suffix)
	}
	return res
}

func (s *Section) CtyType() cty.Type {
	nativeType := reflect.TypeOf(s)
	return cty.Capsule("section", nativeType)
}

func (s *Section) RenderContent(
	ctx context.Context,
	data plugindata.Map,
) (_ plugin.Content, diags diagnostics.Diag) {
	// Section's children are rendered async
	section := new(plugin.ContentSection)
	return section, nil
}

func (s *Section) GetDef() definitions.BlockDef {
	return s.source.Source
}

func (s *Section) isContentTreeEvalBlock() {}

var (
	_ ContentTreeEvalBlock = (*Section)(nil)
	_ RenderableContent    = (*Section)(nil)
)

func LoadSection(
	ctx context.Context,
	providers ContentProviders,
	sectionDef *definitions.Section,
) (_ *Section, diags diagnostics.Diag) {
	// Evaluate requiredVars lists
	var requiredVars []string
	for _, vars := range sectionDef.RequiredVarsCombined {
		var varNames []string
		diag := gohcl.DecodeExpression(vars.Expr, nil, &varNames)
		if diags.Extend(diag) {
			continue
		}
		requiredVars = append(requiredVars, varNames...)
	}

	// Evaluate dependsOn lists
	var dependsOn []EvalKey
	for _, deps := range sectionDef.DependsOnCombined {
		var depNames []string
		diag := gohcl.DecodeExpression(deps.Expr, nil, &depNames)
		if diags.Extend(diag) {
			continue
		}
		for _, name := range depNames {
			defKey, err := definitions.KeyFromName(name)
			if err != nil {
				diags.Extend(diagnostics.FromErr(err))
				return nil, diags
			}
			evalKey := EvalKeyFromDefKey(*defKey)
			dependsOn = append(dependsOn, evalKey)
		}
	}

	block := &Section{
		blockName:    sectionDef.BlockName,
		meta:         sectionDef.Meta,
		vars:         sectionDef.Vars,
		source:       sectionDef,
		requiredVars: requiredVars,
		dependsOn:    dependsOn,
	}
	var diag diagnostics.Diag
	isIncluded := sectionDef.IsIncluded
	if isIncluded == nil {
		isIncluded = defaultIsIncluded(sectionDef.Source.Block.DefRange())
	}

	block.isIncluded, diag = dataspec.DecodeAttr(
		deferred.WithQueryFuncs(appctx.GetEvalContext(ctx)),
		isIncluded,
		isIncludedSpec,
	)
	if diags.Extend(diag) {
		return
	}

	if sectionDef.Title != nil {
		title, diag := LoadContent(ctx, providers, sectionDef.Title)
		if diags.Extend(diag) {
			return
		}
		block.title = title
	}

	for _, child := range sectionDef.Content {
		decoded, diag := LoadContent(ctx, providers, child)
		if diags.Extend(diag) {
			return
		}
		block.children = append(block.children, decoded)
	}
	return block, diags
}
