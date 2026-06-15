// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package constraint

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// lock constraints to their expected values
// changing these would be backwards incompatible (for protobuf)
func TestConstraints(t *testing.T) {
	assert := assert.New(t)
	assert.EqualValues(1, Required)
	assert.EqualValues(2, NonNull)
	assert.EqualValues(4, NonEmpty)
	assert.EqualValues(8, TrimSpace)
	assert.EqualValues(16, Integer)
}
