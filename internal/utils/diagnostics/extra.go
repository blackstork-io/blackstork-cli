package diagnostics

import "github.com/hashicorp/hcl/v2"

type extraList []any

func getExtra[T any](extra any) (_ T, _ bool) {
	for extra != nil {
		if list, ok := extra.(extraList); ok {
			for _, extra := range list {
				val, found := getExtra[T](extra)
				if found {
					return val, found
				}
			}
			return
		}
		if val, ok := extra.(T); ok {
			return val, true
		}
		if val, ok := extra.(hcl.DiagnosticExtraUnwrapper); ok {
			extra = val.UnwrapDiagnosticExtra()
			continue
		}
		return
	}
	return
}

// GetExtra finds extra of type T in a provided diagnostic
func GetExtra[T any](diag *hcl.Diagnostic) (extra T, found bool) {
	return getExtra[T](diag.Extra)
}

// DiagnosticsGetExtra finds the first extra of type T in the slice of provided diagnostics
func DiagnosticsGetExtra[T any](diags Diag) (extra T, found bool) {
	for _, diag := range diags {
		if extra, found = getExtra[T](diag.Extra); found {
			return
		}
	}
	return
}

func FindByExtra[T any](diags Diag) *hcl.Diagnostic {
	for _, diag := range diags {
		if _, found := getExtra[T](diag.Extra); found {
			return diag
		}
	}
	return nil
}

// Adds extra without replacing existing extras.
func addExtraFunc(diag *hcl.Diagnostic, extra any) {
	switch extraT := diag.Extra.(type) {
	case nil:
		diag.Extra = extra
	case extraList:
		extraT = append(extraT, extra)
		diag.Extra = extraT
	default:
		diag.Extra = extraList{extraT, extra}
	}
}
