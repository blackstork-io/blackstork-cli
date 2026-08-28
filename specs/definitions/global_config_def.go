// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

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

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

const patternName = "expose_env_vars_with_pattern"

var DefaultEnvVarsPattern = glob.MustCompile("FABRIC_*")

type GlobalConfigDef struct {
	block *hclsyntax.Block
}

func (g *GlobalConfigDef) GetHCLBlock() *hclsyntax.Block {
	return g.block
}

func (g *GlobalConfigDef) Parse(
	ctx context.Context,
	evalCtx *hcl.EvalContext,
) (cfg *GlobalConfig, diags diagnostics.Diag) {
	var globalCfg GlobalConfig
	var diag diagnostics.Diag

	globalCfg.EnvVarsPattern, diag = g.parseEnvVarPattern(ctx)
	diags.Extend(diag)
	diags.Extend(gohcl.DecodeBody(g.block.Body, evalCtx, &globalCfg))

	if diags.HasErrors() {
		return cfg, diags
	}
	return &globalCfg, diags
}

func (g *GlobalConfigDef) parseEnvVarPattern(
	ctx context.Context,
) (pat glob.Glob, diags diagnostics.Diag) {
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
		return pat, diags
	}
	if attrVal.IsNull() {
		return pat, diags
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
		return pat, diags
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
	return pat, diags
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
