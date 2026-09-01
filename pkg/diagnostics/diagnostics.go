// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

// Package diagnostics provides an ergonomic wrapper around hcl.Diagnostics.
package diagnostics

import (
	"errors"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type Diag hcl.Diagnostics // Diagnostics does implement error interface, but is not, itself, an error.

func (d Diag) Error() string {
	return hcl.Diagnostics(d).Error()
}

// Append adds a diagnostic and reports whether it is an error.
func (d *Diag) Append(diag *hcl.Diagnostic) (addedErrors bool) {
	if diag != nil {
		*d = append(*d, diag)
		return diag.Severity == hcl.DiagError
	}
	return false
}

// Add appends a new error diagnostic.
func (d *Diag) Add(summary, detail string) {
	*d = append(*d, &hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  summary,
		Detail:   detail,
	})
}

// AddWarn appends a new warning diagnostic.
func (d *Diag) AddWarn(summary, detail string) {
	*d = append(*d, &hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  summary,
		Detail:   detail,
	})
}

// Extend appends diagnostics and reports whether they contain an error.
func (d *Diag) Extend(diags []*hcl.Diagnostic) (haveAddedErrors bool) {
	*d = append(*d, diags...)
	return hcl.Diagnostics(diags).HasErrors()
}

// HasErrors returns true if the receiver contains any diagnostics of
// severity DiagError.
func (d Diag) HasErrors() bool {
	return hcl.Diagnostics(d).HasErrors()
}

// AppendErr converts and appends a non-nil error.
func (d *Diag) AppendErr(err error, summary string) (haveAddedErrors bool) {
	// The body of the function is moved into `appendErr` to convince golang to inline the
	// `AppendErr`, making `err != nil` as cheap as usual.
	// Otherwise each AppendErr would waste a slow golang call just to check that err == nil and
	// return false
	haveAddedErrors = err != nil
	if haveAddedErrors {
		appendErr(d, err, summary)
	}
	return haveAddedErrors
}

// Refine applies refiners and returns the diagnostics for chaining.
func (d Diag) Refine(refiners ...Refiner) Diag {
	for _, option := range refiners {
		option.Refine(d)
	}
	return d
}

// AppendErr and appendErr together can't be inlined. We're forbidding go from inlining
// appendErr into AppendErr and thus preventing AppendErr inlining.
//
//go:noinline
func appendErr(d *Diag, err error, summary string) {
	d.Extend(FromErr(err, DefaultSummary(summary)))
}

// FromHCL converts an HCL diagnostic to Diag.
func FromHCL(diag *hcl.Diagnostic) Diag {
	if diag != nil {
		return Diag{diag}
	}
	return nil
}

// FromErr converts an error to Diag.
func FromErr(err error, refiners ...Refiner) (diags Diag) {
	if err == nil {
		return nil
	}
	var diag *hcl.Diagnostic
	var hclDiags hcl.Diagnostics
	switch {
	case errors.As(err, &diags):
	case errors.As(err, &hclDiags):
		diags = Diag(hclDiags)
	case errors.As(err, &diag):
		diags = Diag{diag}
	default:
		diags = Diag{{
			Severity: hcl.DiagError,
			Detail:   err.Error(),
		}}
	}
	if e, ok := errors.AsType[cty.PathError](err); ok {
		refiners = append(refiners, AddPath(e.Path))
	}
	refiners = append(refiners, DefaultSummary("Error"))
	diags.Refine(refiners...)
	return diags
}
