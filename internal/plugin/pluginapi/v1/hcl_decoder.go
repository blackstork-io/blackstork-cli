package pluginapiv1

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/fabric/internal/utils"
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
