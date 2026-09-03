// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package parser

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestFindFiles(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	fs := fstest.MapFS{
		"f1.blackstork.hcl":        &fstest.MapFile{},
		"f2.BLACKSTORK.HCL":        &fstest.MapFile{},
		"f3.not_blackstork":        &fstest.MapFile{},
		"subdir/f4.blackstork.hcl": &fstest.MapFile{},
		"subdir/f5.not_blackstork": &fstest.MapFile{},
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
				"f1.blackstork.hcl",
				"subdir/f4.blackstork.hcl",
			},
		},
		{
			name:      "Non-recursive",
			recursive: false,
			expected: []string{
				"f1.blackstork.hcl",
			},
		},
	}
	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var res []string

			res, diags := findTemplateFiles(fs, tc.recursive)

			assert.Equal(
				tc.expected,
				res,
			)
			assert.Empty(diags)
		})
	}
}
