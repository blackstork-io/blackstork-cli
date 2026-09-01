// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package diagnostics

import "github.com/hashicorp/hcl/v2"

type repeatedError struct{}

// RepeatedError is invisible to users and signals that the initial block evaluation
// has failed (and already has reported its errors to user).
var RepeatedError = &hcl.Diagnostic{
	Severity: hcl.DiagError,
	Extra:    repeatedError{},
}
