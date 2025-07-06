package diagnostics

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type PathExtra struct {
	path    cty.Path
	prepend bool
}

// Refine implements DiagnosticRefiner.
func (p *PathExtra) Refine(diags Diag) {
	for _, d := range diags {
		oldPath, found := getExtra[*PathExtra](d.Extra)
		if found {
			if p.prepend {
				oldPath.path = append(p.path.Copy(), oldPath.path...)
			}
			continue
		}
		pathCopy := *p
		addExtraFunc(d, &pathCopy)
	}
}

func (p *PathExtra) String() string {
	if p == nil || len(p.path) == 0 {
		return ""
	}
	var sb []byte
	for _, step := range p.path {
		switch step := step.(type) {
		case cty.GetAttrStep:
			sb = fmt.Appendf(sb, ".%s", step.Name)
			continue
		case cty.IndexStep:
			ty := step.Key.Type()
			if ty.IsPrimitiveType() && !step.Key.IsNull() && step.Key.IsKnown() {
				if ty == cty.String {
					sb = fmt.Appendf(sb, "[%q]", step.Key.AsString())
					continue
				} else if ty == cty.Number {
					i, _ := step.Key.AsBigFloat().Int64()
					sb = fmt.Appendf(sb, "[%d]", i)
					continue
				}
			}
			sb = fmt.Appendf(sb, "[%s]", step.Key.GoString())
			continue
		}
		sb = fmt.Appendf(sb, "[%+v]", step)
	}
	return string(sb)
}

func (p *PathExtra) improveDiagnostic(diag *hcl.Diagnostic) {
	if p == nil || len(p.path) == 0 {
		return
	}
	diag.Detail = fmt.Sprintf("%s\nHappened while evaluating value at:\n%s", diag.Detail, p)
}

// Stores path in hcl.Diagnostic.Extra, or via Diag.Refine.
// Add path, prepending if any already exist.
func AddPath(p cty.Path) *PathExtra {
	return &PathExtra{
		path:    p.Copy(),
		prepend: true,
	}
}

// Adds path to diagnostics that do not have a path already
func DefaultPath(p cty.Path) *PathExtra {
	return &PathExtra{
		path: p.Copy(),
	}
}
