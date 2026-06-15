// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package appctx

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
