package fabctx

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

func Test_EnvVars(t *testing.T) {
	assert := assert.New(t)
	t.Setenv("TEST_KEY", "test_value")
	evalCtx := newEvalContext()
	env := evalCtx.Variables["env"]
	assert.NotNil(env)
	assert.True(cty.Map(cty.String).Equals(env.Type()))
	envMap := env.AsValueMap()
	assert.True(envMap["NON_EXISTENT_KEY"].IsNull())
	assert.False(envMap["TEST_KEY"].IsNull())
	assert.Equal("test_value", envMap["TEST_KEY"].AsString())
}

func TestFromFileFunc(t *testing.T) {
	const fileContents = "test file contents"
	assert := assert.New(t)
	tmp := t.TempDir()
	tmpPath := path.Join(tmp, "test")
	os.WriteFile(tmpPath, []byte(fileContents), 0o600)
	val, err := fromFileFunc.Call([]cty.Value{cty.StringVal(tmpPath)})
	assert.NoError(err)
	assert.Equal(fileContents, val.AsString())
}

func TestFuncsPresent(t *testing.T) {
	assert := assert.New(t)
	evalCtx := newEvalContext()
	assert.Contains(evalCtx.Functions, "from_file")
	assert.Contains(evalCtx.Functions, "join")
}
