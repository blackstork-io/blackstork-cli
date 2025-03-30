package eval

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

type asyncDataEvaluator struct {
	ctx    context.Context
	blocks []*PluginDataAction
	logger *slog.Logger
}

func makeAsyncDataEvaluator(ctx context.Context, doc *Document, logger *slog.Logger) *asyncDataEvaluator {
	return &asyncDataEvaluator{
		ctx:    ctx,
		blocks: doc.DataBlocks,
		logger: logger,
	}
}

func makeAsyncDataEvaluatorWithPath(
	ctx context.Context,
	doc *Document,
	path []string,
	logger *slog.Logger,
) *asyncDataEvaluator {
	var dataSourceName string
	if len(path) > 0 {
		dataSourceName = path[0]
	}

	var blockName string
	if len(path) > 1 {
		blockName = path[1]
	}

	matchingBlocks := []*PluginDataAction{}

	for i := range doc.DataBlocks {
		block := doc.DataBlocks[i]

		if dataSourceName != "" && block.PluginName != dataSourceName {
			continue
		}

		if blockName != "" && block.BlockName != blockName {
			continue
		}

		matchingBlocks = append(matchingBlocks, block)
	}

	return &asyncDataEvaluator{
		ctx:    ctx,
		blocks: matchingBlocks,
		logger: logger,
	}
}

type asyncDataEvalResult struct {
	pluginName string
	blockName  string
	data       plugindata.Data
	diags      diagnostics.Diag
}

func (doc *asyncDataEvaluator) Execute() (plugindata.Data, diagnostics.Diag) {
	doc.logger.DebugContext(doc.ctx, "Fetching data for the document template")

	resultch := make(chan *asyncDataEvalResult, len(doc.blocks))
	for _, block := range doc.blocks {
		go func(block *PluginDataAction, resultch chan<- *asyncDataEvalResult) {
			doc.logger.DebugContext(
				doc.ctx, "Fetching data for block",
				"plugin", block.PluginName,
				"block", block.BlockName,
			)
			data, diags := block.FetchData(doc.ctx)
			resultch <- &asyncDataEvalResult{
				pluginName: block.PluginName,
				blockName:  block.BlockName,
				data:       data,
				diags:      diags,
			}
		}(block, resultch)
	}

	result := make(plugindata.Map)
	diags := diagnostics.Diag{}

	for i := 0; i < len(doc.blocks); i++ {
		res := <-resultch
		for _, diag := range res.diags {
			diags.Append(diag)
		}

		if diags.HasErrors() {
			return nil, res.diags
		}

		var dsMap plugindata.Map

		if found, ok := result[res.pluginName]; ok {
			dsMap, ok = found.(plugindata.Map)
			if !ok {
				return nil, diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  "Conflicting data",
					Detail: fmt.Sprintf(
						"Different type data block with the same name already exists at plugin '%s' and block '%s'",
						res.pluginName,
						res.blockName,
					),
				}}
			}
		} else {
			dsMap = make(plugindata.Map)
			result[res.pluginName] = dsMap
		}

		dsMap[res.blockName] = res.data
	}

	return result, diags
}

type asyncContent struct {
	ctx       context.Context
	doc       *plugin.ContentSection
	parent    *plugin.ContentSection
	contentID uint32
	content   *Content
	section   *Section
	wg        *sync.WaitGroup
	dependsOn []*asyncContent
}

func (ac *asyncContent) Wait() {
	// Add a timeout to avoid deadlocks in circular reference situations
	done := make(chan struct{})
	go func() {
		ac.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Wait completed normally
	case <-time.After(5 * time.Minute):
		// Deadlock detected or taking too long
		fmt.Printf("Warning: Wait timeout on content block, possibly a circular dependency\n")
	}
}

func (ac *asyncContent) Execute(dataCtx plugindata.Map) diagnostics.Diag {
	defer ac.wg.Done()
	for _, dep := range ac.dependsOn {
		dep.Wait()
	}
	dataCtx = dataCtx.Clone()

	// Create or get the dependency map in the context
	var dependencyMap plugindata.Map
	if depData, exists := dataCtx["dependency"]; exists {
		dependencyMap = depData.(plugindata.Map)
	} else {
		dependencyMap = make(plugindata.Map)
		dataCtx["dependency"] = dependencyMap
	}

	// Add all dependencies to the context for easy access
	for _, dep := range ac.dependsOn {
		// Extract block type, plugin name and block name from dep
		var blockType, pluginName, blockName string

		if dep.content != nil && dep.content.Plugin != nil {
			blockType = definitions.BlockKindContent
			pluginName = dep.content.Plugin.PluginName
			blockName = dep.content.Plugin.BlockName

			// For ref blocks, use "ref" as the plugin name
			if dep.content.Plugin.Source != nil && dep.content.Plugin.Source.IsRef() {
				pluginName = definitions.PluginTypeRef
			}
		} else if dep.section != nil {
			blockType = definitions.BlockKindSection
			// For ref blocks, use "ref" as the plugin name
			if dep.section.source != nil && dep.section.source.IsRef() {
				pluginName = definitions.PluginTypeRef
				blockName = dep.section.source.Name()
			}
		}

		// Skip if we couldn't determine the path
		if blockType == "" || pluginName == "" || blockName == "" {
			continue
		}

		// Get or create the block type map
		var blockTypeMap plugindata.Map
		if btData, exists := dependencyMap[blockType]; exists {
			blockTypeMap = btData.(plugindata.Map)
		} else {
			blockTypeMap = make(plugindata.Map)
			dependencyMap[blockType] = blockTypeMap
		}

		// Get or create the plugin type map
		var pluginTypeMap plugindata.Map
		if ptData, exists := blockTypeMap[pluginName]; exists {
			pluginTypeMap = ptData.(plugindata.Map)
		} else {
			pluginTypeMap = make(plugindata.Map)
			blockTypeMap[pluginName] = pluginTypeMap
		}

		// Store the dependency content based on its type
		if blockType == definitions.BlockKindContent {
			// For content blocks
			if dep.contentID > 0 {
				// Get content from the parent
				parentData := dep.parent.AsData()
				if parentData != nil {
					parentChildren, ok := parentData.(plugindata.Map)["children"]
					if ok {
						childrenArray, ok := parentChildren.(plugindata.List)
						if ok && int(dep.contentID) <= len(childrenArray) {
							pluginTypeMap[blockName] = childrenArray[dep.contentID-1]
						}
					}
				}
			}
		} else if blockType == definitions.BlockKindSection {
			// For sections
			sectionData := plugindata.Map{}

			// Extract metadata from the section
			if metaData := extractSectionMetadata(dep.section); metaData != nil {
				sectionData[definitions.BlockKindMeta] = metaData
			}

			// For reference sections, handle title extraction
			if pluginName == definitions.PluginTypeRef && dep.section != nil {
				// Ensure we have a meta map
				if _, exists := sectionData[definitions.BlockKindMeta]; !exists {
					sectionData[definitions.BlockKindMeta] = plugindata.Map{}
				}

				// Extract title from reference
				if title := extractSectionTitleFromRef(dep.section); title != "" {
					sectionData[definitions.BlockKindMeta].(plugindata.Map)["title"] = plugindata.String(title)
				}
			}

			// Add children if available
			if children := getSectionChildren(dep); children != nil {
				sectionData["children"] = children
			}

			// Store the section data if we have any content
			if len(sectionData) > 0 {
				pluginTypeMap[blockName] = sectionData
			}
		}
	}

	if ac.section != nil {
		diags := ac.section.PrepareData(ac.ctx, dataCtx, ac.doc, ac.parent)
		if diags.HasErrors() {
			return diags
		}
	}

	daigs := ac.content.RenderContent(ac.ctx, dataCtx, ac.doc, ac.parent, ac.contentID)
	return daigs
}

// makeAsyncContent creates a new asyncContent for rendering
func makeAsyncContent(
	ctx context.Context,
	doc *plugin.ContentSection,
	parent *plugin.ContentSection,
	content *Content,
	dependsOn []*asyncContent,
	section *Section,
) *asyncContent {
	tmp := new(plugin.ContentEmpty)
	_ = parent.Add(tmp, nil)

	contentID := tmp.ID()
	wg := new(sync.WaitGroup)
	wg.Add(1)
	return &asyncContent{
		ctx:       ctx,
		doc:       doc,
		parent:    parent,
		contentID: contentID,
		content:   content,
		wg:        wg,
		dependsOn: dependsOn,
		section:   section,
	}
}

// makeAsyncContentForRef creates a special asyncContent for reference blocks
// that doesn't participate in the actual dependency graph execution
func makeAsyncContentForRef(
	ctx context.Context,
	doc *plugin.ContentSection,
	parent *plugin.ContentSection,
	content *Content,
	section *Section,
) *asyncContent {
	// For reference blocks, we don't want to create dependencies
	// They're just placeholders in the namedMap for lookups
	wg := new(sync.WaitGroup)
	// Mark it as done immediately so it doesn't block
	wg.Add(1)
	wg.Done()

	return &asyncContent{
		ctx:       ctx,
		doc:       doc,
		parent:    parent,
		contentID: 0, // Not needed for references
		content:   content,
		wg:        wg,
		dependsOn: nil,
		section:   section,
	}
}

type asyncContentEvaluator struct {
	invokeMap map[plugin.InvocationOrder][]*asyncContent
	namedMap  map[string]*asyncContent
	rootNode  *plugin.ContentSection
}

func (ace *asyncContentEvaluator) executeGroup(
	dataCtx plugindata.Map,
	invokeOrder plugin.InvocationOrder,
) diagnostics.Diag {
	list, ok := ace.invokeMap[invokeOrder]
	if !ok || len(list) == 0 {
		return nil
	}

	diagch := make(chan diagnostics.Diag, len(list))

	for _, ac := range ace.invokeMap[invokeOrder] {
		go func(ac *asyncContent, dataCtx plugindata.Map, diagch chan<- diagnostics.Diag) {
			diagch <- ac.Execute(dataCtx)
		}(ac, dataCtx, diagch)
	}

	diags := diagnostics.Diag{}

	for i := 0; i < len(list); i++ {
		diags.Extend(<-diagch)
	}

	return diags
}

func (ace *asyncContentEvaluator) Execute(dataCtx plugindata.Map) (*plugin.ContentSection, diagnostics.Diag) {
	order := []plugin.InvocationOrder{
		plugin.InvocationOrderBegin,
		plugin.InvocationOrderUnspecified,
		plugin.InvocationOrderEnd,
	}

	diags := diagnostics.Diag{}

	for _, o := range order {
		diags.Extend(ace.executeGroup(dataCtx, o))
		if diags.HasErrors() {
			return nil, diags
		}
	}

	ace.rootNode.Compact()

	return ace.rootNode, diags
}

func makeAsyncContentEvaluator(
	ctx context.Context,
	content []*Content,
	dataCtx plugindata.Map,
) (*asyncContentEvaluator, diagnostics.Diag) {
	namedMap := make(map[string]*asyncContent)
	invokeMap := make(map[plugin.InvocationOrder][]*asyncContent)
	rootNode := plugin.NewSection(0)

	diags := diagnostics.Diag{}

	// First pass: register all reference blocks
	for _, c := range content {
		diag := registerReferences(ctx, c, rootNode, rootNode, namedMap)
		if diags.Extend(diag) {
			return nil, diag
		}
	}

	// Second pass: process content and dependencies
	for _, c := range content {
		diag := assignAsyncContent(ctx, dataCtx, c, rootNode, rootNode, namedMap, invokeMap, nil)
		if diags.Extend(diag) {
			return nil, diag
		}
	}

	return &asyncContentEvaluator{
		invokeMap: invokeMap,
		namedMap:  namedMap,
		rootNode:  rootNode,
	}, diags
}

// registerReferences pre-registers reference blocks to make them available
// for dependency resolution, even if they appear in the template after
// blocks that depend on them.
func registerReferences(
	ctx context.Context,
	c *Content,
	rootNode *plugin.ContentSection,
	parent *plugin.ContentSection,
	namedMap map[string]*asyncContent,
) diagnostics.Diag {
	switch {
	case c.Plugin != nil:
		// Check if this is a content reference (ref block)
		if c.Plugin.Source != nil && c.Plugin.Source.IsRef() {
			ac := makeAsyncContentForRef(ctx, rootNode, parent, c, nil)
			refName := strings.Join([]string{
				definitions.BlockKindContent,
				definitions.PluginTypeRef,
				c.Plugin.BlockName,
			}, ".")
			namedMap[refName] = ac
		}

	case c.Section != nil:
		// Register section references
		if c.Section.source != nil && c.Section.source.IsRef() && c.Section.source.Name() != "" {
			ac := makeAsyncContentForRef(ctx, rootNode, parent, c, c.Section)
			refName := strings.Join([]string{
				definitions.BlockKindSection,
				definitions.PluginTypeRef,
				c.Section.source.Name(),
			}, ".")
			namedMap[refName] = ac
		}
	}

	return nil
}

func assignAsyncContent(
	ctx context.Context,
	dataCtx plugindata.Map,
	c *Content,
	rootNode *plugin.ContentSection,
	parent *plugin.ContentSection,
	namedMap map[string]*asyncContent,
	invokeMap map[plugin.InvocationOrder][]*asyncContent,
	section *Section,
) diagnostics.Diag {
	diags := diagnostics.Diag{}
	switch {
	case c.Plugin != nil:
		var dependsOn []*asyncContent
		if len(c.Plugin.DependsOn) > 0 {
			for _, depName := range c.Plugin.DependsOn {
				dep, ok := namedMap[depName]
				if !ok {
					return diagnostics.Diag{{
						Severity: hcl.DiagError,
						Summary:  "Dependency not found",
						Detail: fmt.Sprintf(
							"Content block '%s' depends on '%s' but it's not found",
							c.Plugin.BlockName,
							depName,
						),
					}}
				}

				dependsOn = append(dependsOn, dep)
			}
		}

		ac := makeAsyncContent(ctx, rootNode, parent, c, dependsOn, section)

		// For all content blocks, register with the standard format: content.pluginName.blockName
		name := strings.Join([]string{
			definitions.BlockKindContent,
			c.Plugin.PluginName,
			c.Plugin.BlockName,
		}, ".")
		namedMap[name] = ac

		// Register references if not already registered in the first pass
		if c.Plugin.Source != nil && c.Plugin.Source.IsRef() {
			refName := strings.Join([]string{
				definitions.BlockKindContent,
				definitions.PluginTypeRef,
				c.Plugin.BlockName,
			}, ".")
			if _, exists := namedMap[refName]; !exists {
				namedMap[refName] = ac
			}
		}

		invokeMap[c.Plugin.Provider.InvocationOrder] = append(invokeMap[c.Plugin.Provider.InvocationOrder], ac)
	case c.Section != nil:
		// Create a section in the content tree
		tmp := plugin.NewSection(0)
		_ = parent.Add(tmp, nil)

		// Register section references if not already registered in the first pass
		if c.Section.source != nil && c.Section.source.IsRef() && c.Section.source.Name() != "" {
			refName := strings.Join([]string{
				definitions.BlockKindSection,
				definitions.PluginTypeRef,
				c.Section.source.Name(),
			}, ".")

			if _, exists := namedMap[refName]; !exists {
				ac := makeAsyncContentForRef(ctx, rootNode, parent, c, c.Section)
				namedMap[refName] = ac
			}
		}

		// Process the section content
		include, children, diag := c.Section.Unwrap(ctx, dataCtx)
		if diags.Extend(diag) || !include {
			return diag
		}

		// Process all child content blocks
		sectionDataCtx := dataCtx.Clone()
		for _, child := range children {
			diag := assignAsyncContent(ctx, sectionDataCtx, child, rootNode, tmp, namedMap, invokeMap, c.Section)
			if diags.Extend(diag) {
				return diag
			}
		}
	default:
		return diagnostics.Diag{{
			Severity: hcl.DiagError,
			Summary:  "Invalid content",
			Detail:   "Content block must be either a plugin or a section",
		}}
	}

	return diags
}

// extractSectionMetadata extracts metadata from a section, including title and other metadata fields
func extractSectionMetadata(section *Section) plugindata.Map {
	if section == nil || section.meta == nil {
		return nil
	}

	metaData := section.meta.AsPluginData().(plugindata.Map)

	// Use the 'name' field from metadata as 'title' for easier access
	if title, ok := metaData["name"]; ok {
		metaData["title"] = title
	}

	return metaData
}

// extractSectionTitleFromRef extracts title from a section reference block
func extractSectionTitleFromRef(section *Section) string {
	if section == nil || section.source == nil || !section.source.IsRef() || section.source.Block == nil {
		return ""
	}

	// First try to get title from direct attribute
	if attr, exists := section.source.Block.Body.Attributes["title"]; exists {
		if expr, ok := attr.Expr.(*hclsyntax.LiteralValueExpr); ok {
			if expr.Val.Type() == cty.String {
				return expr.Val.AsString()
			}
		}
	}

	// If no direct title, try to get it from base reference
	if baseAttr, hasBase := section.source.Block.Body.Attributes["base"]; hasBase {
		if traversalExpr, ok := baseAttr.Expr.(*hclsyntax.ScopeTraversalExpr); ok {
			if len(traversalExpr.Traversal) >= 2 {
				if rootName, ok := traversalExpr.Traversal[0].(hcl.TraverseRoot); ok && rootName.Name == "section" {
					if nameAttr, ok := traversalExpr.Traversal[1].(hcl.TraverseAttr); ok {
						return nameAttr.Name
					}
				}
			}
		}
	}

	return ""
}

// getSectionChildren gets children data from a section's parent
func getSectionChildren(dep *asyncContent) plugindata.Data {
	if dep.contentID == 0 {
		return nil
	}

	parentData := dep.parent.AsData()
	if parentData == nil {
		return nil
	}

	parentChildren, ok := parentData.(plugindata.Map)["children"]
	if !ok {
		return nil
	}

	childrenArray, ok := parentChildren.(plugindata.List)
	if !ok || int(dep.contentID) > len(childrenArray) {
		return nil
	}

	return childrenArray[dep.contentID-1]
}
