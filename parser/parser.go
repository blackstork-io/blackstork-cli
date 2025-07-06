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

	"github.com/blackstork-io/fabric/cmd/fabctx"
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
	file   *hcl.File
	path   string
	blocks *DefinedBlocks
}

func parseHclBytes(ctx context.Context, bytes []byte, path string) (res *fileParseResult, err error) {

	log := fabctx.GetLog(ctx)
	log = log.With("path", path)

	file, err := hclsyntax.ParseConfig(bytes, path, hcl.InitialPos)
	if err != nil {
		log.ErrorContext(ctx, "Error while parsing HCL file", "err", err)
		return nil, err
	}
	res = &fileParseResult{
		file: file,
		path: path,
	}
	body := utils.ToHclsyntaxBody(res.file.Body)
	blocks, err := parseBlockDefinitions(ctx, body)

	if err != nil {
		log.ErrorContext(ctx, "Error while parsing HCL blocks", "err", err)
		return nil, err
	}

	res.blocks = blocks
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
) (*DefinedBlocks, map[string]*hcl.File, diagnostics.Diag) {

	var diags diagnostics.Diag

	templateFiles, readDiags := findTemplateFiles(dir, true)

	if len(templateFiles) == 0 {
		log.WarnContext(ctx, "No templates found", "dir_path", dir)
		return nil, nil, diagnostics.Diag{{
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
			return parseHclBytes(ctx, body, path)
		},
	)

	if len(errs) > 0 {
		log.ErrorContext(ctx, "Errors while parsing files", "errs", errs)
	}

	blocks := NewDefinedBlocks()
	fileMap := map[string]*hcl.File{}

	for _, result := range parseResults {
		blocks.merge(result.blocks)
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

func parseBlockDefinitions(ctx context.Context, body *hclsyntax.Body) (res *DefinedBlocks, diags diagnostics.Diag) {

	log := fabctx.GetLog(ctx)

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

			if origBlock, found := res.execBlockDefs[key]; found {
				kind := origBlock.Kind()
				origDefRange := origBlock.GetHCLBlock().DefRange()
				diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Duplicate `%s` declaration", kind),
					Detail: fmt.Sprintf(
						"`%s` with the same name defined at %s:%d",
						kind,
						origDefRange.Filename,
						origDefRange.Start.Line,
					),
					Subject: execBlock.GetHCLBlock().DefRange().Ptr(),
				})
			} else {
				res.execBlockDefs[key] = execBlock
			}

		case definitions.BlockKindDocument:
			blk, dgs := definitions.DefineDocumentDef(block)
			if diags.Extend(dgs) {
				continue
			}
			diag := AddIfMissing(res.DocumentDefs, blk.Name, blk)
			diags.Append(diag)
		case definitions.BlockKindSection:
			blk, dgs := definitions.DefineSectionDef(block, true)
			if diags.Extend(dgs) {
				continue
			}
			diag := AddIfMissing(res.SectionDefs, blk.Name(), blk)
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
			diags.Append(AddIfMissing(res.ConfigDefs, *key, cfg))
			diags.Append(diag)
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
