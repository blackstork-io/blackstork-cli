package definitions

import (
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
)

const LocalVarName = "local"

type Vars struct {
	// stored in the order of definition
	attrs           []*dataspec.Attr
	attrIndexByName map[string]int
}

func (vrs *Vars) Empty() bool {
	return vrs == nil || len(vrs.attrs) == 0
}

// MergeWithBaseVars handles merging with vars from ref base.
// Shadowing has different rules, and will be handled at the evaluation stage.
// func (vrs *Vars) MergeWithBaseVars(baseVars *Vars) *Vars {
// 	if vrs.Empty() {
// 		return baseVars
// 	}
// 	if baseVars.Empty() {
// 		return vrs
// 	}
//
// 	bVars := slices.Clone(baseVars.attrs)
// 	bVarsattrIndexByName := maps.Clone(baseVars.attrIndexByName)
// 	for _, v := range vrs.attrs {
// 		if idx, found := bVarsattrIndexByName[v.Name]; found {
// 			// redefine, but keep the definition order
// 			bVars[idx] = v
// 		} else {
// 			bVarsattrIndexByName[v.Name] = len(bVars)
// 			bVars = append(bVars, v)
// 		}
// 	}
// 	return &Vars{
// 		attrs: bVars,
// 		attrIndexByName:    bVarsattrIndexByName,
// 	}
// }

// AppendVar appends a variable to the vars struct, last in the evaluation order.
func (vrs *Vars) Append(vars ...*dataspec.Attr) {
	if len(vrs.attrs) == 0 {
		vrs.attrIndexByName = make(map[string]int)
	}
	for _, v := range vars {
		vrs.attrs = append(vrs.attrs, v)
		vrs.attrIndexByName[v.Name] = len(vrs.attrs) - 1
	}
}

func (vrs *Vars) Extend(vars *Vars) *Vars {
	if vars == nil {
		return vrs
	}
	vrs.Append(vars.attrs...)
	return vrs
}

func (vrs *Vars) GetAttrs() []*dataspec.Attr {
	return vrs.attrs
}
