package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlugin_Schema(t *testing.T) {
	schema := Plugin("1.2.3", nil)
	assert.Equal(t, "blackstork/github", schema.Name)
	assert.Equal(t, "1.2.3", schema.Version)
	assert.NotNil(t, schema.DataSources["github_issues"])
	assert.NotNil(t, schema.Publishers["github_gist"])
}
