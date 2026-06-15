// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package parser_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/blackstork-io/blackstork-cli/parser"
)

func TestFindFiles(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	fs := fstest.MapFS{
		"f1.fabric":            &fstest.MapFile{},
		"f2.fAbRiC":            &fstest.MapFile{},
		"f1.blackstork.hcl":    &fstest.MapFile{},
		"f3.not_fabric":        &fstest.MapFile{},
		"subdir/f4.fAbRiC":     &fstest.MapFile{},
		"subdir/f5.not_fabric": &fstest.MapFile{},
	}

	type testCase struct {
		name      string
		recursive bool
		expected  []string
	}

	testCases := []testCase{
		{
			name:      "Recursive",
			recursive: true,
			expected: []string{
				"f1.fabric",
				"f2.fAbRiC",
				"subdir/f4.fAbRiC",
			},
		},
		{
			name:      "Non-recursive",
			recursive: false,
			expected: []string{
				"f1.fabric",
				"f2.fAbRiC",
			},
		},
	}
	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var res []string

			diags := parser.FindFabricFiles(fs, tc.recursive, func(path string) {
				res = append(res, path)
			})

			assert.Equal(
				tc.expected,
				res,
			)
			assert.Empty(diags)
		})
	}
}
