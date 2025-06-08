package parser

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/utils"
)

// Collection of block definitions.
type DefinedBlocks struct {
	GlobalConfig  *definitions.GlobalConfigDef
	ConfigDefs    map[definitions.Key]*definitions.ConfigDef
	DocumentDefs  map[string]*definitions.DocumentDef
	SectionDefs   map[string]*definitions.SectionDef
	ExecBlockDefs map[definitions.Key]*definitions.ExecBlockDef
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

	content, data, format, publish := execBlocksMapToCty(db.ExecBlockDefs)
	cfgContent, cfgData, cfgFormat, cfgPublish := execBlocksMapToCty(db.ConfigDefs)

	config := cty.ObjectVal(map[string]cty.Value{
		definitions.BlockKindData:    cfgData,
		definitions.BlockKindContent: cfgContent,
		definitions.BlockKindFormat:  cfgFormat,
		definitions.BlockKindPublish: cfgPublish,
	})

	var sections cty.Value
	if len(db.SectionDefs) == 0 {
		sections = cty.MapValEmpty(cty.Map((*definitions.SectionDef)(nil).CtyType()))
	} else {
		sect := make(map[string]cty.Value, len(db.SectionDefs))
		for k, v := range db.SectionDefs {
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

func (db *DefinedBlocks) DefaultConfigFor(plugin *definitions.ExecBlockDef) (config *definitions.ConfigDef) {
	return db.DefaultConfigDef(plugin.Kind(), plugin.Name())
}

func (db *DefinedBlocks) DefaultConfigDef(blockKind, runnerName string) (config *definitions.ConfigDef) {
	return db.ConfigDefs[definitions.Key{
		Kind:   blockKind,
		Runner: runnerName,
		Name:   "",
	}]
}

func (db *DefinedBlocks) Merge(other *DefinedBlocks) (diags diagnostics.Diag) {
	if other.GlobalConfig != nil {
		if db.GlobalConfig != nil {
			diags.Add("Global config declared multiple times", "")
		}
		db.GlobalConfig = other.GlobalConfig
	}
	for k, v := range other.ConfigDefs {
		diags.Append(AddIfMissing(db.ConfigDefs, k, v))
	}
	for k, v := range other.DocumentDefs {
		diags.Append(AddIfMissing(db.DocumentDefs, k, v))
	}
	for k, v := range other.SectionDefs {
		diags.Append(AddIfMissing(db.SectionDefs, k, v))
	}
	for k, v := range other.ExecBlockDefs {
		diags.Append(AddIfMissing(db.ExecBlockDefs, k, v))
	}
	return
}

func AddIfMissing[M ~map[K]V, K comparable, V definitions.BlockDef](m M, key K, newBlock V) *hcl.Diagnostic {
	if origBlock, found := m[key]; found {
		kind := origBlock.GetHCLBlock().Type
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
		ConfigDefs:    map[definitions.Key]*definitions.ConfigDef{},
		DocumentDefs:  map[string]*definitions.DocumentDef{},
		SectionDefs:   map[string]*definitions.SectionDef{},
		ExecBlockDefs: map[definitions.Key]*definitions.ExecBlockDef{},
	}
}
