// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package pluginapiv1

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/blackstork-cli/pkg/utils"
)

func encodeDiagnosticList(src []*hcl.Diagnostic) []*Diagnostic {
	return utils.FnMap(src, encodeDiagnostic)
}

func encodeDiagnostic(src *hcl.Diagnostic) *Diagnostic {
	if src == nil {
		return nil
	}
	return &Diagnostic{
		Severity: int64(src.Severity),
		Summary:  src.Summary,
		Detail:   src.Detail,
		Subject:  encodeRange(src.Subject),
		Context:  encodeRange(src.Context),
	}
}

func encodePos(src hcl.Pos) *Pos {
	return &Pos{
		Line:   int64(src.Line),
		Column: int64(src.Column),
		Byte:   int64(src.Byte),
	}
}

func encodeRange(src *hcl.Range) *Range {
	if src == nil {
		return nil
	}
	return &Range{
		Filename: src.Filename,
		Start:    encodePos(src.Start),
		End:      encodePos(src.End),
	}
}

func encodeRangeVal(src hcl.Range) *Range {
	return encodeRange(&src)
}
