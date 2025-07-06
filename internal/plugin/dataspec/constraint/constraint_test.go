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
