package definitions

import (
	"context"
	"fmt"
	"strings"

	"github.com/gobwas/glob"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/utils"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
)

const patternName = "expose_env_vars_with_pattern"

var DefaultEnvVarsPattern = glob.MustCompile("FABRIC_*")

type GlobalConfigDef struct {
	block *hclsyntax.Block
}

func (g *GlobalConfigDef) GetHCLBlock() *hclsyntax.Block {
	return g.block
}

func (g *GlobalConfigDef) Parse(ctx context.Context) (cfg *GlobalConfig, diags diagnostics.Diag) {
	var globalCfg GlobalConfig
	var diag diagnostics.Diag

	evalCtx := fabctx.GetEvalContext(ctx)

	globalCfg.EnvVarsPattern, diag = g.parseEnvVarPattern(ctx)
	diags.Extend(diag)
	diags.Extend(gohcl.DecodeBody(g.block.Body, evalCtx, &globalCfg))

	if diags.HasErrors() {
		return
	}
	return &globalCfg, diags
}

func (g *GlobalConfigDef) parseEnvVarPattern(ctx context.Context) (pat glob.Glob, diags diagnostics.Diag) {
	attr, found := utils.Pop(g.block.Body.Attributes, patternName)
	if !found {
		return DefaultEnvVarsPattern, nil
	}
	defer func() {
		if diags.HasErrors() {
			pat = nil
		}
		diags.Refine(diagnostics.DefaultSubject(attr.Expr.Range()))
	}()

	attrVal, diag := dataspec.DecodeAndEvalAttr(ctx, attr, &dataspec.AttrSpec{
		Name: patternName,
		Type: cty.String,
	}, nil)

	if diags.Extend(diag) {
		return
	}
	if attrVal.IsNull() {
		return
	}
	strVal := attrVal.AsString()

	trimmedStr := strings.TrimSpace(strVal)
	if trimmedStr != strVal {
		diags.AddWarn(
			fmt.Sprintf("%q contains a whitespace", patternName),
			"Leading and trailing whitespaces are ignored",
		)
	}
	if trimmedStr == "" {
		return
	}
	var err error
	pat, err = glob.Compile(trimmedStr)
	if err != nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Failed to parse %q", patternName),
			Detail:   err.Error(),
		})
	}
	return
}

func DefineGlobalConfig(block *hclsyntax.Block) (config *GlobalConfigDef, diags diagnostics.Diag) {
	return &GlobalConfigDef{
		block: block,
	}, nil
}

type GlobalConfig struct {
	CacheDir       string            `hcl:"cache_dir,optional"`
	PluginRegistry *PluginRegistry   `hcl:"plugin_registry,block"`
	PluginVersions map[string]string `hcl:"plugin_versions,optional"`
	EnvVarsPattern glob.Glob
}

type PluginRegistry struct {
	BaseURL   string `hcl:"base_url,optional"`
	MirrorDir string `hcl:"mirror_dir,optional"`
}

func (g *GlobalConfig) Merge(other *GlobalConfig) {
	if other.CacheDir != "" {
		g.CacheDir = other.CacheDir
	}
	if other.PluginRegistry != nil {
		if g.PluginRegistry == nil {
			g.PluginRegistry = other.PluginRegistry
		} else {
			if other.PluginRegistry.BaseURL != "" {
				g.PluginRegistry.BaseURL = other.PluginRegistry.BaseURL
			}
			if other.PluginRegistry.MirrorDir != "" {
				g.PluginRegistry.MirrorDir = other.PluginRegistry.MirrorDir
			}
		}
	}
	if other.EnvVarsPattern != DefaultEnvVarsPattern {
		g.EnvVarsPattern = other.EnvVarsPattern
	}
	g.PluginVersions = other.PluginVersions
}
