package diagnostics

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type Refiner interface {
	Refine(diags Diag)
}

// Set Summary field if empty
type DefaultSummary string

func (ds DefaultSummary) Refine(diags Diag) {
	for _, d := range diags {
		if d.Summary == "" {
			d.Summary = string(ds)
		}
	}
}

// Set Subject field if empty
type DefaultSubject hcl.Range

func (ds DefaultSubject) Refine(diags Diag) {
	for _, d := range diags {
		if d.Subject == nil {
			d.Subject = (*hcl.Range)(&ds)
		}
	}
}

// Adds an extra without replacing existing extras.
func AddExtra(extra any) Refiner {
	switch eT := extra.(type) {
	case *PathExtra:
		return eT
	case cty.Path:
		return AddPath(eT)
	default:
		return &extraAdder{eT}
	}
}

type extraAdder struct {
	extra any
}

func (ae *extraAdder) Refine(diags Diag) {
	for _, d := range diags {
		addExtraFunc(d, ae.extra)
	}
}
