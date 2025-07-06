package fabctx

import (
	"context"
	"log/slog"
	"maps"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/joho/godotenv"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

// Creates a new eval context. Base eval context is cloned and extended with environment variables.
func newEvalContext() *hcl.EvalContext {
	vars := maps.Clone(baseEvalContext.Variables)
	if vars == nil {
		vars = make(map[string]cty.Value)
	}
	vars["env"] = buildEnvVarMap()
	evalCtx := &hcl.EvalContext{
		Variables: vars,
		Functions: maps.Clone(baseEvalContext.Functions),
	}
	return evalCtx
}

func buildEnvVarMap() cty.Value {
	envVars := os.Environ()
	envFromFile, err := godotenv.Read()
	if err != nil && !os.IsNotExist(err) {
		slog.Error("Error reading .env file", "err", err)
	}
	envMap := make(map[string]cty.Value, len(envFromFile)+len(envVars))
	for k, v := range envFromFile {
		envMap[k] = cty.StringVal(v)
	}
	for _, e := range envVars {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) != 2 {
			continue
		}
		envMap[pair[0]] = cty.StringVal(pair[1])
	}
	return cty.MapVal(envMap)
}

var fromFileFunc = function.New(&function.Spec{
	Description: "Reads the content of a file and returns it as a string",
	Params: []function.Parameter{{
		Name:        "path",
		Description: "The path to the file to read",
		Type:        cty.String,
	}},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		res, err := os.ReadFile(args[0].AsString())
		if err != nil {
			return cty.NullVal(cty.String), err
		}
		return cty.StringVal(string(res)), nil
	},
})

var baseEvalContext = &hcl.EvalContext{
	Functions: map[string]function.Function{
		"from_file": fromFileFunc,
		"join":      stdlib.JoinFunc,
	},
}

type evalCtxKeyT struct{}

var evalCtxKey = evalCtxKeyT{}

func GetEvalContext(ctx context.Context) *hcl.EvalContext {
	if ctx != nil {
		if ec, ok := ctx.Value(evalCtxKey).(*hcl.EvalContext); ok {
			return ec
		}
	}
	// slog.InfoContext(ctx, "No eval context found, using base eval context")
	return baseEvalContext
}

func WithEvalContext(ctx context.Context, evalCtx *hcl.EvalContext) context.Context {
	return context.WithValue(ctx, evalCtxKey, evalCtx)
}
