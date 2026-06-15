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
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetAt(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name  string
		slice []int
		idx   int
		val   int
	}

	tests := []testCase{
		{
			name:  "Regular append",
			slice: []int{0, 1, 2},
			idx:   3,
			val:   1337,
		},
		{
			name:  "Set existing",
			slice: []int{0, 1, 2},
			idx:   1,
			val:   1337,
		},
		{
			name:  "Set and extend",
			slice: []int{0, 1, 2},
			idx:   10,
			val:   1337,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert := assert.New(t)
			orig := slices.Clone(tc.slice)
			res := SetAt(tc.slice, tc.idx, tc.val)

			assert.Len(res, max(len(orig), tc.idx+1))

			for i := range res {
				switch {
				case i == tc.idx: // value at `idx` should become `val`
					assert.Equal(tc.val, res[i])
				case i < len(orig): // other values from the original slice shouldn't change
					assert.Equal(orig[i], res[i])
				default: // i >= len(orig)
					// other values should be filled by the zero value
					assert.Zero(res[i])
				}
			}
		})
	}
}

func TestFnMap(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	assert.Equal([]string{"1", "2", "3"}, FnMap([]int{1, 2, 3}, strconv.Itoa))
	assert.Equal([]string{}, FnMap([]int{}, strconv.Itoa))
}
