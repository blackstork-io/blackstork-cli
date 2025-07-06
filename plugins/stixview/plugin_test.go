package stixview

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlugin_Schema(t *testing.T) {
	schema := Plugin("1.2.3")
	assert.Equal(t, "blackstork/stixview", schema.Name)
	assert.Equal(t, "1.2.3", schema.Version)
	assert.NotNil(t, schema.ContentProviders["stixview"])
}
