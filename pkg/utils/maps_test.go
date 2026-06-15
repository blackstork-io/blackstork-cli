// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPop(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	var m map[string]int
	v, found := Pop(m, "d")
	assert.Zero(v)
	assert.False(found)

	m = map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}
	v, found = Pop(m, "d")
	assert.Zero(v)
	assert.False(found)

	assert.Contains(m, "a")
	v, found = Pop(m, "a")
	assert.Equal(1, v)
	assert.True(found)

	assert.Equal(
		map[string]int{
			"b": 2,
			"c": 3,
		},
		m,
	)
}

func TestSliceToSet(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	var slice []string

	assert.Equal(map[string]struct{}{}, SliceToSet(slice))

	slice = []string{"a", "b", "c", "c", "b"}

	assert.Equal(map[string]struct{}{
		"a": {},
		"b": {},
		"c": {},
	}, SliceToSet(slice))
}
