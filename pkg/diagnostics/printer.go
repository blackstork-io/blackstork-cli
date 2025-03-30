package diagnostics

import (
	"bufio"
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

	// Convert width to uint safely to avoid potential overflow
	var diagWidth uint
	if width > 0 {
		diagWidth = uint(width)
	} else {
		diagWidth = 80 // Default width
	}

	diagWriter := hcl.NewDiagnosticTextWriter(bufWr, fileMap, diagWidth, colorize)

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
