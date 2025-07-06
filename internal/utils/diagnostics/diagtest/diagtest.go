package diagtest

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/utils"
)

type Assert interface {
	Assert(diag *hcl.Diagnostic) bool
	fmt.Stringer
}

type severityAssert struct {
	severity hcl.DiagnosticSeverity
}

func (s severityAssert) Assert(diag *hcl.Diagnostic) bool {
	return diag.Severity == s.severity
}

// String implements Assert.
func (s severityAssert) String() string {
	switch s.severity {
	case hcl.DiagError:
		return "An error"
	case hcl.DiagWarning:
		return "A warning"
	default:
		return fmt.Sprintf("Severity to be equal to hcl.DiagnosticSeverity(%d)", s.severity)
	}
}

var _ Assert = (*severityAssert)(nil)

var (
	IsError   = severityAssert{hcl.DiagError}
	IsWarning = severityAssert{hcl.DiagWarning}
)

type containsAssert struct {
	isSummary bool // if false - detail
	substrs   []string
}

func (c *containsAssert) Assert(diag *hcl.Diagnostic) bool {
	var str string
	if c.isSummary {
		str = diag.Summary
	} else {
		str = diag.Detail
	}
	str = strings.ToLower(str)
	for _, substr := range c.substrs {
		if !strings.Contains(str, strings.ToLower(substr)) {
			return false
		}
	}
	return true
}

// String implements Assert.
func (c *containsAssert) String() string {
	var attrName string
	if c.isSummary {
		attrName = "Summary"
	} else {
		attrName = "Detail"
	}

	return fmt.Sprintf(
		"%s to contain: %s",
		attrName,
		strings.Join(
			utils.FnMap(
				c.substrs,
				func(s string) string {
					return fmt.Sprintf("%q", s)
				},
			),
			", ",
		),
	)
}

func SummaryContains(substrs ...string) Assert {
	return &containsAssert{
		isSummary: true,
		substrs:   substrs,
	}
}

func DetailContains(substrs ...string) Assert {
	return &containsAssert{
		isSummary: false,
		substrs:   substrs,
	}
}

type equalsAssert struct {
	isSummary bool // if false - detail
	str       string
}

func (e *equalsAssert) Assert(diag *hcl.Diagnostic) bool {
	var str string
	if e.isSummary {
		str = diag.Summary
	} else {
		str = diag.Detail
	}
	return str == e.str
}

// String implements Assert.
func (e *equalsAssert) String() string {
	var attrName string
	if e.isSummary {
		attrName = "Summary"
	} else {
		attrName = "Detail"
	}

	return fmt.Sprintf(
		"%s to be equal to: %s",
		attrName,
		e.str,
	)
}

func SummaryEquals(str string) Assert {
	return &equalsAssert{
		isSummary: true,
		str:       str,
	}
}

func DetailEquals(str string) Assert {
	return &equalsAssert{
		isSummary: false,
		str:       str,
	}
}

type Asserts [][]Assert

func formatCondensedDiag(sb *strings.Builder, d *hcl.Diagnostic) {
	switch d.Severity {
	case hcl.DiagError:
		sb.WriteString("    [Severity]: Error\n")
	case hcl.DiagWarning:
		sb.WriteString("    [Severity]: Warning\n")
	case hcl.DiagInvalid:
		sb.WriteString("    [Severity]: Invalid\n")
	default:
		fmt.Fprintf(sb, "    [Severity]: hcl.DiagnosticSeverity(%d)\n", d.Severity)
	}
	fmt.Fprintf(sb, "    [Summary]: %s\n", d.Summary)
	fmt.Fprintf(sb, "    [Detail]: %s\n", d.Detail)
}

func (asserts Asserts) AssertMatch(tb testing.TB, diags []*hcl.Diagnostic, fileMap map[string]*hcl.File) {
	tb.Helper()
	unmatchedDiags, unmatchedAsserts := matchBiject(diags, asserts)
	if len(unmatchedDiags) == 0 && len(unmatchedAsserts) == 0 {
		return
	}
	var buf strings.Builder

	if len(unmatchedAsserts) != 0 {
		buf.WriteString("Unmatched asserts:\n")
		for _, assertSet := range unmatchedAsserts {
			fmt.Fprintln(&buf, "{")
			for _, assert := range assertSet {
				fmt.Fprintf(&buf, "    %s\n", assert)
			}
			fmt.Fprintln(&buf, "},")
		}
	}

	if len(unmatchedDiags) != 0 {
		buf.WriteString("Unmatched diags:\n")
		if len(fileMap) != 0 {
			// we have filemap, so we use fancy print that shows line ranges
			diagnostics.PrintDiags(&buf, diags, fileMap, false)
		} else {
			// condensed format
			for _, diag := range unmatchedDiags {
				buf.WriteString("{\n")
				formatCondensedDiag(&buf, diag)
				buf.WriteString("},\n")
			}
		}
	}
	tb.Fatal(buf.String())
}

func sliceRemove[T any](s []T, pos int) []T {
	var tmp T
	s[pos] = s[len(s)-1]
	s[len(s)-1] = tmp
	return s[:len(s)-1]
}

func matchBiject(d []*hcl.Diagnostic, a [][]Assert) (unmatchedDiags []*hcl.Diagnostic, unmatchedAsserts [][]Assert) {
	unmatchedDiags = slices.Clone(d)
	unmatchedAsserts = slices.Clone(a)

nextDiag:
	for diagIdx := 0; diagIdx < len(unmatchedDiags); {
	nextAssertSet:
		for assertSetIdx, assertSet := range unmatchedAsserts {
			if len(assertSet) == 0 {
				panic("assert set has length 0")
			}
			for _, assert := range assertSet {
				if !assert.Assert(unmatchedDiags[diagIdx]) {
					// This assert set doesn't match current diag
					continue nextAssertSet
				}
			}
			// all asserts in assert set have matched, remove both assert set and diag
			unmatchedAsserts = sliceRemove(unmatchedAsserts, assertSetIdx)
			unmatchedDiags = sliceRemove(unmatchedDiags, diagIdx)
			// intentionally not incrementing diagIdx, sliceRemove replaces the deleted element with the last one
			continue nextDiag
		}
		// can't find an assert set matching this diag
		diagIdx += 1
	}
	return
}

func AssertNoErrors(tb testing.TB, diags []*hcl.Diagnostic, fileMap map[string]*hcl.File, msgs ...any) {
	tb.Helper()
	if len(diags) == 0 {
		return
	}
	var buf bytes.Buffer
	diagnostics.PrintDiags(&buf, diags, fileMap, false)
	msgs = append(msgs, buf.String())
	if hcl.Diagnostics(diags).HasErrors() {
		tb.Error(msgs...)
	} else {
		tb.Log(msgs...)
	}
}
