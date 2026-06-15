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

func decodeDiagnosticList(src []*Diagnostic) []*hcl.Diagnostic {
	return utils.FnMap(src, decodeDiagnostic)
}

func decodeDiagnostic(src *Diagnostic) *hcl.Diagnostic {
	if src == nil {
		return nil
	}
	return &hcl.Diagnostic{
		Severity: hcl.DiagnosticSeverity(src.GetSeverity()),
		Summary:  src.GetSummary(),
		Detail:   src.GetDetail(),
		Subject:  decodeRange(src.GetSubject()).Ptr(),
		Context:  decodeRange(src.GetContext()).Ptr(),
	}
}

func decodePos(src *Pos) hcl.Pos {
	if src == nil {
		return hcl.InitialPos
	}
	return hcl.Pos{
		Line:   int(src.GetLine()),
		Column: int(src.GetColumn()),
		Byte:   int(src.GetByte()),
	}
}

func decodeRange(src *Range) hcl.Range {
	if src == nil {
		return hcl.Range{
			Filename: "<missing range info>",
			Start:    hcl.InitialPos,
			End:      hcl.InitialPos,
		}
	}
	return hcl.Range{
		Filename: src.GetFilename(),
		Start:    decodePos(src.GetStart()),
		End:      decodePos(src.GetEnd()),
	}
}
