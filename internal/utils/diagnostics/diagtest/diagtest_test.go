package diagtest

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"

	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
)

func TestAssertMatch(t *testing.T) {
	diags := []*hcl.Diagnostic{
		{
			Severity: hcl.DiagError,
			Summary:  "Error summary",
			Detail:   "Error detail",
		},
		{
			Severity: hcl.DiagWarning,
			Summary:  "Warning summary",
			Detail:   "Warning detail",
		},
	}

	asserts := Asserts{
		{
			IsError,
			SummaryEquals("Error summary"),
			SummaryContains("Error", "summary"),
			DetailEquals("Error detail"),
			DetailContains("Error", "detail"),
		},
		{
			IsWarning,
			SummaryEquals("Warning summary"),
			SummaryContains("Warning", "summary"),
			DetailEquals("Warning detail"),
			DetailContains("Warning", "detail"),
		},
	}

	asserts.AssertMatch(t, diags, nil)
}

func TestMatchBiject(t *testing.T) {
	testCases := []struct {
		desc             string
		diags            diagnostics.Diag
		asserts          Asserts
		unmatchedDiags   []int
		unmatchedAsserts []int
	}{
		{
			desc: "SimpleNoMatch",
			diags: diagnostics.Diag{
				{
					Severity: hcl.DiagError,
					Summary:  "Error summary",
				},
			},
			asserts: Asserts{
				{IsError, SummaryEquals("Another error summary")},
			},
			unmatchedDiags:   []int{0},
			unmatchedAsserts: []int{0},
		},
		{
			desc: "PartialMatch",
			diags: diagnostics.Diag{
				{
					Severity: hcl.DiagError,
					Summary:  "Error summary",
				},
				{
					Severity: hcl.DiagError,
					Summary:  "This is a match",
				},
			},
			asserts: Asserts{
				{IsError, SummaryEquals("Another error summary")},
				{IsError, SummaryEquals("This is a match")},
			},
			unmatchedDiags:   []int{0},
			unmatchedAsserts: []int{0},
		},
		{
			desc: "PartialMatchOrderDoesNotMatter",
			diags: diagnostics.Diag{
				{
					Severity: hcl.DiagError,
					Summary:  "This is a match",
				},
				{
					Severity: hcl.DiagError,
					Summary:  "Error summary",
				},
			},
			asserts: Asserts{
				{IsError, SummaryEquals("Another error summary")},
				{IsError, SummaryEquals("This is a match")},
			},
			unmatchedDiags:   []int{1},
			unmatchedAsserts: []int{0},
		},
		{
			desc: "ComplexPartialMatch",
			diags: diagnostics.Diag{
				{
					Severity: hcl.DiagError,
					Summary:  "0",
				},
				{
					Severity: hcl.DiagError,
					Summary:  "1",
				},
				{
					Severity: hcl.DiagError,
					Summary:  "2",
				},
				{
					Severity: hcl.DiagError,
					Summary:  "3",
				},
			},
			asserts: Asserts{
				{IsError, SummaryEquals("3")},
				{IsError, SummaryEquals("6")},
				{IsError, SummaryEquals("1")},
				{IsError, SummaryEquals("5")},
			},
			unmatchedDiags:   []int{0, 2},
			unmatchedAsserts: []int{1, 3},
		},
		{
			desc: "ComplexFullMatch",
			diags: diagnostics.Diag{
				{
					Severity: hcl.DiagError,
					Summary:  "0",
				},
				{
					Severity: hcl.DiagError,
					Summary:  "1",
				},
				{
					Severity: hcl.DiagError,
					Summary:  "2",
				},
				{
					Severity: hcl.DiagError,
					Summary:  "3",
				},
			},
			asserts: Asserts{
				{IsError, SummaryEquals("3")},
				{IsError, SummaryEquals("1")},
				{IsError, SummaryEquals("0")},
				{IsError, SummaryEquals("2")},
			},
			unmatchedDiags:   []int{},
			unmatchedAsserts: []int{},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			unmatchedDiags, unmatchedAsserts := matchBiject(tC.diags, tC.asserts)
			var expectedUnmatchedDiags []*hcl.Diagnostic
			for _, i := range tC.unmatchedDiags {
				expectedUnmatchedDiags = append(expectedUnmatchedDiags, tC.diags[i])
			}
			var expectedUnmatchedAsserts Asserts
			for _, i := range tC.unmatchedAsserts {
				expectedUnmatchedAsserts = append(expectedUnmatchedAsserts, tC.asserts[i])
			}

			assert.ElementsMatch(t, expectedUnmatchedDiags, unmatchedDiags)
			assert.ElementsMatch(t, expectedUnmatchedAsserts, unmatchedAsserts)
		})
	}
}
