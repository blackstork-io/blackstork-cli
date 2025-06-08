package diagnostics

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/mattn/go-colorable"
	"golang.org/x/term"
)

func PrintDiags(output io.Writer, diags []*hcl.Diagnostic, fileMap map[string]*hcl.File, colorize bool) {
	if len(diags) == 0 {
		return
	}
	width := 80
	if file, ok := output.(*os.File); ok {
		termWidth, _, err := term.GetSize(int(file.Fd()))
		if err == nil && termWidth > 0 {
			width = termWidth
		}
		if colorize {
			output = colorable.NewColorable(file)
		}
	}

	bufWr := bufio.NewWriter(output)
	defer func() {
		err := bufWr.Flush()
		if err != nil {
			slog.Error("Failed to flush diagnostics", "err", err)
		}
	}()

	diagWriter := hcl.NewDiagnosticTextWriter(bufWr, fileMap, uint(width), colorize)

	for _, diag := range diags {
		if _, isRepeated := GetExtra[repeatedError](diag); isRepeated {
			continue
		}
		if gojqErr, ok := GetExtra[GoJQError](diag); ok {
			gojqErr.improveDiagnostic(diag, fileMap)
		}
		if traceback, ok := GetExtra[TracebackExtra](diag); ok {
			traceback.improveDiagnostic(diag)
		}
		if path, ok := GetExtra[PathExtra](diag); ok {
			path.improveDiagnostic(diag)
		}
		err := diagWriter.WriteDiagnostic(diag)
		if err != nil {
			slog.Error("Failed to write diagnostics", "err", err)
		}
	}
}

func GetDiagsDetails(diags []*hcl.Diagnostic) (details []map[string]any) {
	if len(diags) == 0 {
		return details
	}

	details = []map[string]any{}

	for _, diag := range diags {
		details = append(details, GetDiagDetails(diag))
	}
	return details
}

func GetDiagDetails(diag *hcl.Diagnostic) (details map[string]any) {
	if diag == nil {
		return details
	}

	details = map[string]any{}

	if diag.Subject != nil {
		details["filename"] = diag.Subject.Filename
	}

	details["summary"] = diag.Summary
	details["details"] = diag.Detail

	severity := ""
	switch diag.Severity {
	case hcl.DiagError:
		severity = "error"
	case hcl.DiagWarning:
		severity = "warning"
	case hcl.DiagInvalid:
		severity = "invalid"
	default:
		severity = fmt.Sprintf("value=%d", diag.Severity)
	}

	details["severity"] = severity

	if gojqErr, ok := GetExtra[GoJQError](diag); ok {
		details["jq_error"] = extractJQErrorDetails(gojqErr)
	}

	if traceback, ok := GetExtra[TracebackExtra](diag); ok {
		details["traceback"] = extractTracebackDetails(traceback)
	}

	if path, ok := GetExtra[PathExtra](diag); ok {
		details["path"] = path.String()
	}
	return details
}
