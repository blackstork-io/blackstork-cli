package definitions

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

const (
	BlockKindDocument     = "document"
	BlockKindConfig       = "config"
	BlockKindContent      = "content"
	BlockKindPublish      = "publish"
	BlockKindData         = "data"
	BlockKindMeta         = "meta"
	BlockKindVars         = "vars"
	BlockKindSection      = "section"
	BlockKindGlobalConfig = "fabric"
	BlockKindDynamic      = "dynamic"
	BlockKindFormat       = "format"

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

// Identifies a defined block
type Key struct {
	Kind string
	// Data source, content provider, formatter or publisher
	Runner string
	Name   string
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
			return nil, fmt.Errorf("invalid block type found in a dependency name `%s`", val)
		}
		kind = BlockKindSection
		name = parts[1]
	} else if len(parts) == 3 {
		kind = parts[0]
		runner = parts[1]
		name = parts[2]
	} else {
		return nil, fmt.Errorf("error parsing a dependency name `%s`", val)
	}

	if !isValidKind(kind) {
		return nil, fmt.Errorf("invalid block type found in a dependency name `%s`", val)
	}

	return &Key{
		Kind:   kind,
		Runner: runner,
		Name:   name,
	}, nil
}
