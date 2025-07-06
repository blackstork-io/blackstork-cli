package diagnostics

import "github.com/hashicorp/hcl/v2"

type repeatedError struct{}

// Invisible to user error, typically used to signal that the initial block evaluation
// has failed (and already has reported its errors to user).
var RepeatedError = &hcl.Diagnostic{
	Severity: hcl.DiagError,
	Extra:    repeatedError{},
}
