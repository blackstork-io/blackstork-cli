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
