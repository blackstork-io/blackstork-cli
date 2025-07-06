package pluginapiv1

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/fabric/internal/utils"
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
