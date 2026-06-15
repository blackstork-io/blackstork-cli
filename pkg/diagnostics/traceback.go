// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package diagnostics

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
)

type TracebackExtra = *tracebackExtra

type tracebackExtra struct {
	Traceback []*hcl.Range
}

func extractTracebackDetails(tb *tracebackExtra) []map[string]any {
	details := []map[string]any{}
	for _, rng := range tb.Traceback {
		var d map[string]any
		if rng != nil {
			d = map[string]any{
				"filename":   rng.Filename,
				"start_line": rng.Start.Line,
				"column":     rng.Start.Column,
			}
		} else {
			d = map[string]any{
				"filename": "<unknown>",
			}
		}
		details = append(details, d)
	}
	return details
}

func (tb *tracebackExtra) improveDiagnostic(diag *hcl.Diagnostic) {
	sb := []byte(diag.Detail)
	for _, rng := range tb.Traceback {
		if rng != nil {
			sb = fmt.Appendf(
				sb, "\n  at %s:%d:%d",
				rng.Filename, rng.Start.Line, rng.Start.Column,
			)
		} else {
			sb = append(sb, "\n  at <missing location info>"...)
		}
	}
	diag.Detail = string(sb)
}

func NewTracebackExtra() TracebackExtra {
	return &tracebackExtra{}
}
