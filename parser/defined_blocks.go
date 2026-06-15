// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package parser

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

type BlocksRegistry interface {
	GetGlobalConfig() *definitions.GlobalConfigDef
	GetDefaultBlockConfig(block *definitions.ExecBlockDef) *definitions.ConfigDef
	GetConfigDefsMap() map[definitions.Key]*definitions.ConfigDef

	ResolveRefBase(base hcl.Expression, target Ctyable) (definitions.BlockDef, diagnostics.Diag)

	GetDocumentDefsMap() map[string]*definitions.DocumentDef
	GetDocumentDefByName(name string) (*definitions.DocumentDef, bool)

	GetSectionDefsMap() map[string]*definitions.SectionDef
	GetSectionDefByName(name string) (*definitions.SectionDef, bool)

	GetExecBlockDefsMap() map[definitions.Key]*definitions.ExecBlockDef
	GetExecBlockDefByKey(key definitions.Key) (*definitions.ExecBlockDef, bool)

	Merge(other BlocksRegistry, override bool) diagnostics.Diag

	IsBlocksRegistry()
}

// Collection of block definitions.
type blocksRegistry struct {
	globalConfig  *definitions.GlobalConfigDef
	configDefs    map[definitions.Key]*definitions.ConfigDef
	documentDefs  map[string]*definitions.DocumentDef
	sectionDefs   map[string]*definitions.SectionDef
	execBlockDefs map[definitions.Key]*definitions.ExecBlockDef
}

func (reg *blocksRegistry) IsBlocksRegistry() {}

func (reg *blocksRegistry) GetGlobalConfig() *definitions.GlobalConfigDef {
	return reg.globalConfig
}

func (reg *blocksRegistry) GetConfigDefsMap() map[definitions.Key]*definitions.ConfigDef {
	return reg.configDefs
}

func (reg *blocksRegistry) GetDefaultBlockConfig(
	blockDef *definitions.ExecBlockDef,
) (config *definitions.ConfigDef) {
	return reg.defaultConfigDef(blockDef.Kind(), blockDef.RunnerName())
}

func (reg *blocksRegistry) GetDocumentDefsMap() map[string]*definitions.DocumentDef {
	return reg.documentDefs
}

func (reg *blocksRegistry) GetDocumentDefByName(name string) (*definitions.DocumentDef, bool) {
	val, ok := reg.documentDefs[name]
	return val, ok
}

func (reg *blocksRegistry) GetSectionDefsMap() map[string]*definitions.SectionDef {
	return reg.sectionDefs
}

func (reg *blocksRegistry) GetSectionDefByName(name string) (*definitions.SectionDef, bool) {
	val, ok := reg.sectionDefs[name]
	return val, ok
}

func (reg *blocksRegistry) GetExecBlockDefsMap() map[definitions.Key]*definitions.ExecBlockDef {
	return reg.execBlockDefs
}

func (reg *blocksRegistry) GetExecBlockDefByKey(key definitions.Key) (*definitions.ExecBlockDef, bool) {
	val, ok := reg.execBlockDefs[key]
	return val, ok
}

func (reg *blocksRegistry) defaultConfigDef(kind, runner string) (config *definitions.ConfigDef) {
	return reg.configDefs[definitions.Key{
		Kind:   kind,
		Runner: runner,
		Name:   "",
	}]
}

func mapGetOrInit[K1, K2 comparable, V any](m map[K1]map[K2]V, key K1) (innerMap map[K2]V) {
	innerMap, found := m[key]
	if !found {
		innerMap = map[K2]V{}
		m[key] = innerMap
	}
	return innerMap
}

func execBlocksMapToCty[V definitions.BlockDef](
	execBlocksMap map[definitions.Key]V,
) (content, data, format, publish cty.Value) {
	// [runner_name][block_name]ctyVal

	contentBlocks := map[string]map[string]cty.Value{}
	dataBlocks := map[string]map[string]cty.Value{}
	formatBlocks := map[string]map[string]cty.Value{}
	publishBlocks := map[string]map[string]cty.Value{}

	for k, v := range execBlocksMap {
		switch k.Kind {
		case definitions.BlockKindContent:
			blockNameToVal := mapGetOrInit(contentBlocks, k.Runner)
			blockNameToVal[k.Name] = definitions.ToCtyValue(v)
		case definitions.BlockKindData:
			blockNameToVal := mapGetOrInit(dataBlocks, k.Runner)
			blockNameToVal[k.Name] = definitions.ToCtyValue(v)
		case definitions.BlockKindFormat:
			blockNameToVal := mapGetOrInit(formatBlocks, k.Runner)
			blockNameToVal[k.Name] = definitions.ToCtyValue(v)
		case definitions.BlockKindPublish:
			blockNameToVal := mapGetOrInit(publishBlocks, k.Runner)
			blockNameToVal[k.Name] = definitions.ToCtyValue(v)
		default:
			panic("Unsupported block kind encountered")
		}
	}

	var contentBlocksCty cty.Value
	if len(contentBlocks) > 0 {
		contentBlocksCty = utils.MapMapToCty(contentBlocks)
	}

	var dataBlocksCty cty.Value
	if len(dataBlocks) > 0 {
		dataBlocksCty = utils.MapMapToCty(dataBlocks)
	}

	var formatBlocksCty cty.Value
	if len(formatBlocks) > 0 {
		formatBlocksCty = utils.MapMapToCty(formatBlocks)
	}

	var publishBlocksCty cty.Value
	if len(publishBlocks) > 0 {
		publishBlocksCty = utils.MapMapToCty(publishBlocks)
	}

	return contentBlocksCty, dataBlocksCty, formatBlocksCty, publishBlocksCty
}

// The map to use in HCL eval context when block definitions are resolved
func (reg *blocksRegistry) AsValueMap() map[string]cty.Value {
	content, data, format, publish := execBlocksMapToCty(reg.execBlockDefs)
	cfgContent, cfgData, cfgFormat, cfgPublish := execBlocksMapToCty(reg.configDefs)

	config := cty.ObjectVal(map[string]cty.Value{
		definitions.BlockKindData:    cfgData,
		definitions.BlockKindContent: cfgContent,
		definitions.BlockKindFormat:  cfgFormat,
		definitions.BlockKindPublish: cfgPublish,
	})

	var sections cty.Value
	if len(reg.sectionDefs) == 0 {
		sections = cty.MapValEmpty(cty.Map((*definitions.SectionDef)(nil).CtyType()))
	} else {
		sect := make(map[string]cty.Value, len(reg.sectionDefs))
		for k, v := range reg.sectionDefs {
			sect[k] = definitions.ToCtyValue(v)
		}
		sections = cty.MapVal(sect)
	}
	return map[string]cty.Value{
		definitions.BlockKindConfig:  config,
		definitions.BlockKindData:    data,
		definitions.BlockKindContent: content,
		definitions.BlockKindSection: sections,
		definitions.BlockKindFormat:  format,
		definitions.BlockKindPublish: publish,
	}
}

func (reg *blocksRegistry) Merge(other BlocksRegistry, override bool) (diags diagnostics.Diag) {
	if other.GetGlobalConfig() != nil {
		if reg.globalConfig != nil {
			diags.Add("Global config declared multiple times", "")
		}
		reg.globalConfig = other.GetGlobalConfig()
	}
	for k, v := range other.GetConfigDefsMap() {
		if override {
			reg.configDefs[k] = v
		} else {
			diags.Append(AddIfMissing(reg.configDefs, k, v))
		}
	}
	for k, v := range other.GetDocumentDefsMap() {
		if override {
			reg.documentDefs[k] = v
		} else {
			diags.Append(AddIfMissing(reg.documentDefs, k, v))
		}
	}
	for k, v := range other.GetSectionDefsMap() {
		if override {
			reg.sectionDefs[k] = v
		} else {
			diags.Append(AddIfMissing(reg.sectionDefs, k, v))
		}
	}
	for k, v := range other.GetExecBlockDefsMap() {
		if override {
			reg.execBlockDefs[k] = v
		} else {
			diags.Append(AddIfMissing(reg.execBlockDefs, k, v))
		}
	}
	return diags
}

func (reg *blocksRegistry) ResolveRefBase(
	expr hcl.Expression,
	result Ctyable,
) (definitions.BlockDef, diagnostics.Diag) {
	blockMap := reg.AsValueMap()
	switch result.(type) {
	case *definitions.ExecBlockDef:
		return Resolve[*definitions.ExecBlockDef](blockMap, expr)
	case *definitions.SectionDef:
		return Resolve[*definitions.SectionDef](blockMap, expr)
	case *definitions.ConfigDef:
		return Resolve[*definitions.ConfigDef](blockMap, expr)
	default:
		var diags diagnostics.Diag
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported ref target block type",
			Detail:   fmt.Sprintf("Error while trying to resolve a ref base expression. Unsupported target block type: `%T`", result),
		})
		return nil, diags
	}
}

func Resolve[B Ctyable](blockMap map[string]cty.Value, expr hcl.Expression) (B, diagnostics.Diag) {
	var res B

	val, diag := expr.Value(&hcl.EvalContext{
		Variables: blockMap,
	})
	var diags diagnostics.Diag
	if diags.Extend(diag) {
		return res, diags
	}
	expectedType := res.CtyType()

	ty := val.Type()
	if !ty.Equals(expectedType) {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incorrect reference",
			Detail: fmt.Sprintf(
				"Expected reference to `%s` but got a reference to `%s`",
				expectedType.FriendlyName(),
				ty.FriendlyName(),
			),
			Subject: expr.Range().Ptr(),
		})
		return res, diags
	}
	res = val.EncapsulatedValue().(B)
	return res, diags
}

func AddIfMissing[M ~map[K]V, K comparable, V definitions.BlockDef](m M, key K, newBlock V) *hcl.Diagnostic {
	if origBlock, found := m[key]; found {
		kind := origBlock.Kind()
		origDefRange := origBlock.GetHCLBlock().DefRange()
		return &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Duplicate `%s` declaration", kind),
			Detail: fmt.Sprintf(
				"`%s` with the same name defined at %s:%d",
				kind,
				origDefRange.Filename,
				origDefRange.Start.Line,
			),
			Subject: newBlock.GetHCLBlock().DefRange().Ptr(),
		}
	}
	m[key] = newBlock
	return nil
}

func NewDefinedBlocks() *blocksRegistry {
	return &blocksRegistry{
		configDefs:    map[definitions.Key]*definitions.ConfigDef{},
		documentDefs:  map[string]*definitions.DocumentDef{},
		sectionDefs:   map[string]*definitions.SectionDef{},
		execBlockDefs: map[definitions.Key]*definitions.ExecBlockDef{},
	}
}

var _ BlocksRegistry = (*blocksRegistry)(nil)
