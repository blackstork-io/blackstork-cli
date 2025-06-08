package parser

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/utils"
)

// FS-level parsing of fabric files

const (
	templateFileExt     = ".fabric"
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
		if strings.EqualFold(filepath.Ext(path), templateFileExt) {
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
	diagnostics.Diag
	file   *hcl.File
	path   string
	blocks *DefinedBlocks
}

func parseHclBytes(bytes []byte, path string) (res *fileParseResult) {
	file, diag := hclsyntax.ParseConfig(bytes, path, hcl.InitialPos)
	res = &fileParseResult{
		file: file,
		path: path,
		Diag: diagnostics.Diag(diag),
	}
	if res.HasErrors() {
		return
	}

	body := utils.ToHclsyntaxBody(res.file.Body)

	blocks, diags := parseBlockDefinitions(body)
	res.Extend(diags)
	res.blocks = blocks

	return res
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
) (*DefinedBlocks, map[string]*hcl.File, diagnostics.Diag) {

	var diags diagnostics.Diag

	templateFiles, readDiags := findTemplateFiles(dir, true)
	log.DebugContext(ctx, "Template files found", "files_count", len(templateFiles), "dir", dir)

	diags = append(diags, readDiags...)

	workersCount := min(len(templateFiles), maxReadParseWorkers)

	parseResults, _ := utils.RunWorkers(
		ctx,
		log.With("task", "parsing_templates"),
		templateFiles,
		workersCount,
		func(path string) (*fileParseResult, error) {
			body, err := readFile(dir, path)
			if err != nil {
				return &fileParseResult{
					Diag: diagnostics.Diag{{
						Severity: hcl.DiagError,
						Summary:  "File read error",
						Detail:   fmt.Sprintf("Error while looking at `%s`: %s", path, err),
						Extra:    err,
					}},
					path: path,
				}, nil
			}
			return parseHclBytes(body, path), nil
		},
	)

	blocks := NewDefinedBlocks()
	fileMap := map[string]*hcl.File{}

	for _, result := range parseResults {
		diags.Extend(result.Diag)
		if result.HasErrors() {
			continue
		}

		blocks.Merge(result.blocks)
		fileMap[result.path] = result.file
	}

	if len(fileMap) == 0 {
		diags.Add(
			"No valid templates files found",
			fmt.Sprintf("No valid template files found in `%s`", dir),
		)
	}

	return blocks, fileMap, diags
}

func parseBlockDefinitions(body *hclsyntax.Body) (res *DefinedBlocks, diags diagnostics.Diag) {
	res = NewDefinedBlocks()

	for _, block := range body.Blocks {
		switch block.Type {
		// Standalone top level outside-document blocks
		case definitions.BlockKindData, definitions.BlockKindContent, definitions.BlockKindFormat, definitions.BlockKindPublish:
			execBlock, dgs := definitions.DefineExecBlockDef(block, true)
			if diags.Extend(dgs) {
				continue
			}
			key := execBlock.Key()
			diags.Append(AddIfMissing(res.ExecBlockDefs, key, execBlock))
		case definitions.BlockKindDocument:
			blk, dgs := definitions.DefineDocumentDef(block)
			if diags.Extend(dgs) {
				continue
			}
			diags.Append(AddIfMissing(res.DocumentDefs, blk.Name, blk))
		case definitions.BlockKindSection:
			blk, dgs := definitions.DefineSectionDef(block, true)
			if diags.Extend(dgs) {
				continue
			}
			diags.Append(AddIfMissing(res.SectionDefs, blk.Name(), blk))
		case definitions.BlockKindConfig:
			cfg, dgs := definitions.DefineConfigDef(block)
			if diags.Extend(dgs) {
				continue
			}
			key := cfg.Key()
			if key == nil {
				panic("unable to get the key of the top-level config block")
			}
			diags.Append(AddIfMissing(res.ConfigDefs, *key, cfg))
		case definitions.BlockKindGlobalConfig:
			if res.GlobalConfig != nil {
				origRng := res.GlobalConfig.GetHCLBlock().DefRange()
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
			res.GlobalConfig = cfg
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
				}))
		}
	}

	return
}
