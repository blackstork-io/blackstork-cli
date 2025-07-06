package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/blackstork-io/fabric/internal/utils"
)

func TestMemoizedKeys(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	type testCase struct {
		name string
		m    map[string]struct{}
		want string
	}
	tests := []testCase{
		{
			name: "None",
			m:    map[string]struct{}{},
			want: "",
		},
		{
			name: "One",
			m: map[string]struct{}{
				"one": {},
			},
			want: "'one'",
		},
		{
			name: "Two",
			m: map[string]struct{}{
				"one": {},
				"two": {},
			},
			want: "'one', 'two'",
		},

		{
			name: "Sorted",
			m: map[string]struct{}{
				"E": {},
				"D": {},
				"C": {},
				"B": {},
				"A": {},
			},
			want: "'A', 'B', 'C', 'D', 'E'",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(tc.want, utils.MemoizedKeys(&tc.m)())
		})
	}
}

func TestMemoizedKeysMemoizes(t *testing.T) {
	assert := assert.New(t)
	m := map[string]struct{}{
		"A": {},
		"B": {},
	}
	mk := utils.MemoizedKeys(&m)
	m["C"] = struct{}{}
	mkRes := mk()
	assert.Equal("'A', 'B', 'C'", mkRes)
	m["D"] = struct{}{}
	mkRes2 := mk()
	assert.Equal("'A', 'B', 'C'", mkRes2)
}
