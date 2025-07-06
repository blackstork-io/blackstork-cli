package parser

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/utils"
)

type BlocksRegistry interface {
	IsBlocksRegistry()

	GetDefaultRunnerConfigForBlock(block *definitions.ExecBlockDef) *definitions.ConfigDef

	ResolveRefBase(base hcl.Expression, target Ctyable) (definitions.BlockDef, diagnostics.Diag)
}

// Collection of block definitions.
type DefinedBlocks struct {
	globalConfig  *definitions.GlobalConfigDef
	configDefs    map[definitions.Key]*definitions.ConfigDef
	documentDefs  map[string]*definitions.DocumentDef
	sectionDefs   map[string]*definitions.SectionDef
	execBlockDefs map[definitions.Key]*definitions.ExecBlockDef
}

func (db *DefinedBlocks) IsBlocksRegistry() {}

func (db *DefinedBlocks) GetDefaultRunnerConfigForBlock(
	blockDef *definitions.ExecBlockDef,
) (config *definitions.ConfigDef) {
	return db.defaultConfigDef(blockDef.Kind(), blockDef.Name())
}

func (db *DefinedBlocks) defaultConfigDef(kind, runner string) (config *definitions.ConfigDef) {
	return db.configDefs[definitions.Key{
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
	return
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
func (db *DefinedBlocks) AsValueMap() map[string]cty.Value {

	content, data, format, publish := execBlocksMapToCty(db.execBlockDefs)
	cfgContent, cfgData, cfgFormat, cfgPublish := execBlocksMapToCty(db.configDefs)

	config := cty.ObjectVal(map[string]cty.Value{
		definitions.BlockKindData:    cfgData,
		definitions.BlockKindContent: cfgContent,
		definitions.BlockKindFormat:  cfgFormat,
		definitions.BlockKindPublish: cfgPublish,
	})

	var sections cty.Value
	if len(db.sectionDefs) == 0 {
		sections = cty.MapValEmpty(cty.Map((*definitions.SectionDef)(nil).CtyType()))
	} else {
		sect := make(map[string]cty.Value, len(db.sectionDefs))
		for k, v := range db.sectionDefs {
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

func (db *DefinedBlocks) merge(other *DefinedBlocks) (diags diagnostics.Diag) {
	if other.globalConfig != nil {
		if db.globalConfig != nil {
			diags.Add("Global config declared multiple times", "")
		}
		db.globalConfig = other.globalConfig
	}
	for k, v := range other.configDefs {
		diags.Append(AddIfMissing(db.configDefs, k, v))
	}
	for k, v := range other.documentDefs {
		diags.Append(AddIfMissing(db.documentDefs, k, v))
	}
	for k, v := range other.sectionDefs {
		diags.Append(AddIfMissing(db.sectionDefs, k, v))
	}
	for k, v := range other.execBlockDefs {
		diags.Append(AddIfMissing(db.execBlockDefs, k, v))
	}
	return
}

func (db *DefinedBlocks) ResolveRefBase(expr hcl.Expression, result Ctyable) (definitions.BlockDef, diagnostics.Diag) {
	blockMap := db.AsValueMap()
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

func NewDefinedBlocks() *DefinedBlocks {
	return &DefinedBlocks{
		configDefs:    map[definitions.Key]*definitions.ConfigDef{},
		documentDefs:  map[string]*definitions.DocumentDef{},
		sectionDefs:   map[string]*definitions.SectionDef{},
		execBlockDefs: map[definitions.Key]*definitions.ExecBlockDef{},
	}
}
