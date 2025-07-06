package parser_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/blackstork-io/fabric/parser"
)

func TestFindFiles(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	fs := fstest.MapFS{
		"f1.fabric":            &fstest.MapFile{},
		"f2.fAbRiC":            &fstest.MapFile{},
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
