// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package parser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

const (
	templateFileExtOld  = ".fabric"
	templateFileExt     = ".blackstork.hcl"
	maxReadParseWorkers = 10
)

// Calls fn with paths to every template file and collects errors into the returned diags.
func findTemplateFiles(rootDir fs.FS, recursive bool) (paths []string, diags diagnostics.Diag) {
	paths = []string{}
	err := fs.WalkDir(rootDir, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Directory traversal error",
				Detail:   fmt.Sprintf("Error while reading `%s`: %s", path, err),
				Extra:    err,
			})
			return nil
		}
		if d.IsDir() {
			if !recursive && path != "." {
				return fs.SkipDir
			}
			return nil
		}
		// Ext() only gets the last part, so we need to check for suffix
		if strings.HasSuffix(path, templateFileExt) {
			paths = append(paths, path)
		}
		// check for .fabric for backward compatibility
		if strings.EqualFold(filepath.Ext(path), templateFileExtOld) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Error while walking a directory",
			Detail:   err.Error(),
			Extra:    err,
		})
	}
	return paths, diags
}

type fileParseResult struct {
	file   *hcl.File
	path   string
	Blocks BlocksRegistry
}

func ParseHclBytes(ctx context.Context, bytes []byte, path string) (*fileParseResult, error) {
	log := appctx.Log(ctx)
	log = log.With("path", path)

	file, diags := hclsyntax.ParseConfig(bytes, path, hcl.InitialPos)
	if diags.HasErrors() {
		log.ErrorContext(ctx, "Error while parsing an HCL file", "err", diags)
		return nil, diags
	}
	res := &fileParseResult{
		file: file,
		path: path,
	}
	body := utils.ToHclsyntaxBody(res.file.Body)

	blocks, diags2 := parseBlockDefinitions(body)
	if diags2.HasErrors() {
		log.ErrorContext(ctx, "Error while parsing HCL blocks", "err", diags)
		return nil, diags
	}

	res.Blocks = blocks
	return res, nil
}

func readFile(rootDir fs.FS, path string) ([]byte, error) {
	file, err := rootDir.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open a file: %w", err)
	}
	defer file.Close()
	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read a file: %w", err)
	}
	return bytes, nil
}

func ParseDir(
	ctx context.Context,
	log *slog.Logger,
	dir fs.FS,
) (BlocksRegistry, map[string]*hcl.File, diagnostics.Diag) {
	var diags diagnostics.Diag

	blocks := NewDefinedBlocks()

	templateFiles, readDiags := findTemplateFiles(dir, true)

	if len(templateFiles) == 0 {
		log.WarnContext(ctx, "No templates found", "dir", dir)
		return blocks, nil, diagnostics.Diag{{
			Severity: hcl.DiagWarning,
			Summary:  "No templates found",
			Detail:   fmt.Sprintf("No templates found in the provided directory `%s`", dir),
		}}
	}

	log.DebugContext(ctx, "Template files found", "count", len(templateFiles), "dir_path", dir)

	diags = append(diags, readDiags...)

	workersCount := min(len(templateFiles), maxReadParseWorkers)

	parseResults, errs := utils.RunWorkers(
		ctx,
		log.With("task", "parsing_templates"),
		templateFiles,
		workersCount,
		func(path string) (*fileParseResult, error) {
			body, err := readFile(dir, path)
			if err != nil {
				log.ErrorContext(ctx, "Error while reading a file", "dir", dir, "path", path)
				return nil, errors.New("error while reading a file")
			}
			return ParseHclBytes(ctx, body, path)
		},
	)

	if len(errs) > 0 {
		log.ErrorContext(ctx, "Errors while parsing files", "errs", errs)
	}

	fileMap := map[string]*hcl.File{}

	for _, result := range parseResults {
		blocks.Merge(result.Blocks, false)
		fileMap[result.path] = result.file
		log.DebugContext(
			ctx, "Blocks found",
			"file", result.path,
			"config_blocks_count", len(result.Blocks.GetConfigDefsMap()),
			"docs_count", len(result.Blocks.GetDocumentDefsMap()),
			"sections_count", len(result.Blocks.GetSectionDefsMap()),
			"exec_blocks_count", len(result.Blocks.GetExecBlockDefsMap()),
		)
	}

	if len(fileMap) == 0 {
		diags.Add(
			"No valid templates files found",
			fmt.Sprintf("No valid template files found in `%s`", dir),
		)
	}

	return blocks, fileMap, diags
}

func parseBlockDefinitions(body *hclsyntax.Body) (res *blocksRegistry, diags diagnostics.Diag) {
	res = NewDefinedBlocks()

	for _, block := range body.Blocks {
		switch block.Type {
		// Standalone top level outside-document blocks
		case definitions.BlockKindData,
			definitions.BlockKindContent,
			definitions.BlockKindFormat,
			definitions.BlockKindPublish:
			execBlock, dgs := definitions.DefineExecBlockDef(block, true)
			if diags.Extend(dgs) {
				continue
			}
			key := execBlock.Key()
			diag := AddIfMissing(res.execBlockDefs, key, execBlock)
			diags.Append(diag)
		case definitions.BlockKindDocument:
			blk, dgs := definitions.DefineDocumentDef(block)
			if diags.Extend(dgs) {
				continue
			}
			diag := AddIfMissing(res.documentDefs, blk.Name, blk)
			diags.Append(diag)
		case definitions.BlockKindSection:
			blk, dgs := definitions.DefineSectionDef(block, true)
			if diags.Extend(dgs) {
				continue
			}
			diag := AddIfMissing(res.sectionDefs, blk.Name(), blk)
			diags.Append(diag)
		case definitions.BlockKindConfig:
			cfg, dgs := definitions.DefineConfigDef(block)
			if diags.Extend(dgs) {
				continue
			}
			key := cfg.Key()
			if key == nil {
				panic("unable to get the key of the top-level config block")
			}
			diag := AddIfMissing(res.configDefs, *key, cfg)
			diags.Append(diag)
		case definitions.BlockKindGlobalConfig, definitions.BlockKindGlobalConfigOld:
			if res.globalConfig != nil {
				origRng := res.globalConfig.GetHCLBlock().DefRange()
				diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Global config redefinition",
					Detail: fmt.Sprintf(
						"Global config must be defined at most once. Original definition at %s:%d",
						origRng.Filename, origRng.Start.Line,
					),
					Subject: block.DefRange().Ptr(),
				})
				continue
			}
			cfg, diag := definitions.DefineGlobalConfig(block)
			if diags.Extend(diag) {
				continue
			}
			res.globalConfig = cfg
		default:
			diags.Append(definitions.NewNestingDiag(
				"Top level of the document template",
				block,
				body,
				[]string{
					definitions.BlockKindData,
					definitions.BlockKindContent,
					definitions.BlockKindPublish,
					definitions.BlockKindDocument,
					definitions.BlockKindFormat,
					definitions.BlockKindSection,
					definitions.BlockKindConfig,
					definitions.BlockKindGlobalConfig,
				},
			))
		}
	}

	return res, diags
}
