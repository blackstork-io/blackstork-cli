package terraform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlugin_Schema(t *testing.T) {
	schema := Plugin("1.2.3")
	assert.Equal(t, "blackstork/terraform", schema.Name)
	assert.Equal(t, "1.2.3", schema.Version)
	assert.NotNil(t, schema.DataSources["terraform_state_local"])
}
