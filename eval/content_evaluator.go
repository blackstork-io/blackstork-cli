// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package eval

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/blackstork-cli/pkg/appctx"
	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/pkg/utils"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugins/builtin"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/definitions"
)

const (
	maxWorkerCount = 20
)

type ExecNode struct {
	id string

	blockID string

	block  RenderableContent
	parent RenderableContent

	isStart bool
	isEnd   bool

	dependencies []*ExecNode
	dependants   []*ExecNode

	wg *sync.WaitGroup
}

func makeID(id string, key EvalKey, parent RenderableContent, isStart, isEnd bool) string {
	name := key.AsName()

	if parent != nil {
		name = fmt.Sprintf("%s-%s", parent.EvalKey().AsName(), name)
	}

	// 	// To make sure the ID is unique
	// 	uniquePrefix := uuid.New().String()[:8]

	prefixedName := fmt.Sprintf("%s-%s", id, name)

	if isStart {
		prefixedName += ":start"
	}
	if isEnd {
		prefixedName += ":end"
	}
	return prefixedName
}

func getDependsOnKeys(block RenderableContent) ([]EvalKey, error) {
	dependencies := []EvalKey{}

	switch block := block.(type) {
	case *Section:
		dependencies = append(dependencies, block.dependsOn...)
	case *PluginContentAction:
		dependencies = append(dependencies, block.dependsOn...)
	default:
		return nil, fmt.Errorf("unexpected block type encoundeted: `%T`", block)
	}
	return dependencies, nil
}

func createExecNodesRecursive(
	ctx context.Context,
	parent RenderableContent,
	block RenderableContent,
	depth int,
) (res []*ExecNode) {
	key := block.EvalKey()

	if block.Kind() == definitions.BlockKindSection {
		// splitting every section node into 2 nodes: start and end nodes
		startNode := &ExecNode{
			id:      makeID(block.ID(), key, parent, true, false),
			blockID: block.ID(),
			block:   block,
			parent:  parent,
			isStart: true,
		}
		endNode := &ExecNode{
			id:      makeID(block.ID(), key, parent, false, true),
			blockID: block.ID(),
			block:   block,
			parent:  parent,
			isEnd:   true,
		}
		res = append(res, startNode, endNode)

		section := block.(*Section)

		if section.titleToRender != nil {
			childNodes := createExecNodesRecursive(ctx, block, section.titleToRender, depth)
			res = append(res, childNodes...)
		}

		children := section.childrenToRender
		for _, child := range children {
			childNodes := createExecNodesRecursive(ctx, block, child, depth+1)
			res = append(res, childNodes...)
		}

	} else {
		node := &ExecNode{
			id:      makeID(block.ID(), key, parent, false, false),
			blockID: block.ID(),
			block:   block,
			parent:  parent,
		}
		res = append(res, node)
	}
	return res
}

func wireNodeDependencies(ctx context.Context, nodes []*ExecNode) ([]*ExecNode, error) {
	log := appctx.Log(ctx)

	for _, node := range nodes {

		_log := log.With("node_id", node.id, "block_id", node.blockID, "block_name", node.block.EvalKey().AsName())

		if node.block.Kind() == definitions.BlockKindSection && node.isEnd {
			// Section end nodes only depend on the children of the section and are wired after all
			// other blocks are hooked in
			continue
		}

		dependsOn, err := getDependsOnKeys(node.block)
		if err != nil {
			_log.ErrorContext(ctx, "Error getting depends on values for block", "err", err)
			return nil, err
		}

		for _, depKey := range dependsOn {
			var depNode *ExecNode

			for _, n := range nodes {
				if n.block.EvalKey() != depKey { // dependencies are by name
					continue
				}

				// if the dependency is a section, depend on its end block
				if depKey.Kind == definitions.BlockKindSection && n.isEnd {
					depNode = n
					break
				}

				// ignore section start nodes in this pass
				if depKey.Kind == definitions.BlockKindSection && n.isStart {
					continue
				}

				depNode = n
			}

			if depNode == nil {
				_log.ErrorContext(ctx, "Dependency not found in the template", "dependency", depKey.AsName())
				return nil, fmt.Errorf("dependency `%s` not found", depKey.AsName())
			}

			node.dependencies = append(node.dependencies, depNode)
			depNode.dependants = append(depNode.dependants, node)
		}

		// if node has a parent, the node must depend on the parent's start node
		if node.parent != nil {
			parentBlockID := node.parent.ID()

			var parentStartNode *ExecNode

			for _, n := range nodes {
				if n.blockID == parentBlockID && n.isStart {
					parentStartNode = n
					break
				}
			}

			if parentStartNode == nil {
				parentName := node.parent.EvalKey().AsName()
				log.ErrorContext(
					ctx, "Parent start node not found in the node list",
					"parent_name", parentName,
					"parent_block_id", parentBlockID,
				)
				return nil, fmt.Errorf("start node for parent block `%s` not found", parentName)
			}

			node.dependencies = append(node.dependencies, parentStartNode)
			parentStartNode.dependants = append(parentStartNode.dependants, node)
		}
	}

	for _, node := range nodes {
		if node.block.Kind() != definitions.BlockKindSection || !node.isEnd {
			continue
		}

		_log := log.With("node_id", node.id, "block_id", node.blockID, "block_name", node.block.EvalKey().AsName())

		var twinStartNode *ExecNode
		for _, n := range nodes {
			if n.blockID == node.blockID && n.isStart {
				twinStartNode = n
				break
			}
		}
		if twinStartNode == nil {
			_log.ErrorContext(ctx, "Sibling start node not found")
			return nil, fmt.Errorf("start node for block `%s` not found", node.block.EvalKey().AsName())
		}

		// Depend on all blocks that wait on the start node
		for _, startDependant := range twinStartNode.dependants {

			// if one of the dependants a subsection start, find it's end
			if startDependant.isStart {
				found := false
				for _, _n := range nodes {
					if _n.blockID == startDependant.blockID && _n.isEnd {
						startDependant = _n
						found = true
						break
					}
				}
				if !found {
					depBlockName := startDependant.block.EvalKey().AsName()
					log.ErrorContext(
						ctx, "End node for dependant block not found",
						"dependant_block_name", depBlockName,
						"dependant_block_id", startDependant.blockID,
					)
					return nil, fmt.Errorf("start node for dependant block `%s` not found", depBlockName)
				}
			}

			node.dependencies = append(node.dependencies, startDependant)
			startDependant.dependants = append(startDependant.dependants, node)
		}
		log.DebugContext(
			ctx,
			"End node wired to wait for children",
			"dependencies", utils.FnMap(node.dependencies, func(d *ExecNode) string { return d.id }),
			"start_node_id", twinStartNode.id,
		)
	}

	return nodes, nil
}

func fillInDependsOnRefs(ctx context.Context, blocks []RenderableContent) ([]*ExecNode, error) {
	log := appctx.Log(ctx)
	log.InfoContext(ctx, "Filling in the depedency references")

	nodes := []*ExecNode{}

	for _, b := range blocks {
		subnodes := createExecNodesRecursive(ctx, nil, b, 0)
		nodes = append(nodes, subnodes...)
	}

	nodeNames := []string{}
	for _, n := range nodes {
		nodeNames = append(nodeNames, n.id)
	}

	log.DebugContext(
		ctx, "Catalog all block eval nodes",
		"blocks_count", len(blocks),
		"nodes_count", len(nodes),
		// "node_names", strings.Join(nodeNames, ", "),
	)

	nodes, err := wireNodeDependencies(ctx, nodes)
	if err != nil {
		return nil, err
	}

	err = detectCycles(ctx, nodes)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func detectCycles(ctx context.Context, nodes []*ExecNode) error {
	log := appctx.Log(ctx)
	log.DebugContext(
		ctx, "Detecting dependency cycles",
		"nodes_count", len(nodes),
	)

	degrees := map[string]int{}
	for _, n := range nodes {
		for _, dependant := range n.dependants {
			degree, ok := degrees[dependant.id]
			if !ok {
				degree = 0
			}
			degrees[dependant.id] = degree + 1
		}
	}

	queue := []*ExecNode{}
	for _, n := range nodes {
		if degrees[n.id] == 0 {
			queue = append(queue, n)
		}
	}

	count := 0
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		count += 1

		var parentKey *EvalKey
		if n.parent != nil {
			parentKey = new(n.parent.EvalKey())
		}

		log.DebugContext(
			ctx, "Collected dependencies for eval node",
			"node", n.id,
			"parent", parentKey,
			"dependencies_count", len(n.dependencies),
			"dependants_count", len(n.dependants),
		)

		for _, dependant := range n.dependants {
			degree, ok := degrees[dependant.id]

			if !ok {
				log.ErrorContext(
					ctx, "No dependency degree value for node",
					"node", dependant.id,
				)
				return fmt.Errorf("no dependency degree value for node `%s`", dependant.id)
			}
			degree -= 1
			degrees[dependant.id] = degree

			if degree <= 0 {
				queue = append(queue, dependant)
			}
		}
	}

	unprocessed := []string{}
	for id, degree := range degrees {
		if degree > 0 {
			unprocessed = append(unprocessed, id)
		}
	}

	if count != len(nodes) {
		log.ErrorContext(
			ctx, "Dependency cycle detected",
			"unreachable_nodes", unprocessed,
		)
		return errors.New("dependency cycle detected")
	}
	return nil
}

func renderContentAsync(ctx context.Context, nodes []*ExecNode) (map[string]plugin.Content, error) {
	log := appctx.Log(ctx)
	log.DebugContext(
		ctx, "Rendering the content",
		"nodes_count", len(nodes),
	)

	degrees := map[string]int{}
	for _, n := range nodes {
		for _, dependant := range n.dependants {
			degree, ok := degrees[dependant.id]
			if !ok {
				degree = 0
			}
			degrees[dependant.id] = degree + 1
		}
	}

	planQueue := []*ExecNode{}
	// prefil planQueue with blocks that has no dependencies
	for _, n := range nodes {
		if degrees[n.id] == 0 {
			planQueue = append(planQueue, n)
		}
	}

	nodesByID := map[string]*ExecNode{}

	runQueue := []*ExecNode{}

	for len(planQueue) > 0 {
		node := planQueue[0]
		planQueue = planQueue[1:]

		// Create new executable content node
		node.wg = new(sync.WaitGroup)
		node.wg.Add(1)

		nodesByID[node.id] = node
		runQueue = append(runQueue, node)

		for _, child := range node.dependants {
			degree, ok := degrees[child.id]

			if !ok {
				log.ErrorContext(ctx, "Incorrect dependency in-degree for node", "node", child.id)
				return nil, fmt.Errorf("incorrect dependency in-degree for node `%s`", child.id)
			}
			degree -= 1
			degrees[child.id] = degree
			if degree <= 0 {
				planQueue = append(planQueue, child)
			}
		}
	}

	unprocessed := []string{}
	for id, degree := range degrees {
		if degree > 0 {
			unprocessed = append(unprocessed, id)
		}
	}

	if len(runQueue) != len(nodes) {
		log.ErrorContext(
			ctx, "Unreachable nodes found, possibly a dependency cycle",
			"unreachable_nodes", unprocessed,
		)
		return nil, fmt.Errorf("unreachable nodes found")
	}

	workerCount := min(maxWorkerCount, len(runQueue))

	resultsMtx := new(sync.Mutex)
	results := map[string]plugin.Content{}

	processedNodes, errs := utils.RunWorkers(
		ctx,
		log.With("task", "render_template_node"),
		runQueue,
		workerCount,
		func(node *ExecNode) (*ExecNode, error) {
			defer node.wg.Done()

			_log := log.With(
				"node_id", node.id,
				"block_id", node.blockID,
				"block_name", node.block.EvalKey().AsName(),
			)
			_log.DebugContext(ctx, "Rendering eval node")

			depIDs := []string{}
			for _, d := range node.dependencies {
				depIDs = append(depIDs, d.id)
			}
			_log.DebugContext(ctx, "Waiting for node dependencies", "deps", depIDs)

			// Actual waiting for dependencies
			for _, dep := range node.dependencies {
				dep.wg.Wait()
			}

			if node.block.Kind() == definitions.BlockKindSection && node.isStart {
				// Section start node has nothing to do, except waiting for the external dependencies
				return node, nil
			}

			if node.block.Kind() == definitions.BlockKindSection && node.isEnd {

				// depth := 0
				// dctx := node.block.GetDataCtx()
				// if dctx != nil {
				// 	if details, ok := dctx[plugin.DetailsDataKey]; ok {
				// 		if val, ok := details[depthDetailsDataContextKey]; ok {
				// 			depth = int(val.(plugindata.Number))
				// 		}
				// 	}
				// }

				evalKey := node.block.EvalKey()
				contentSection := plugin.NewEmptySection(
					&plugin.BlockDetails{
						Name:   evalKey.Name,
						Runner: evalKey.Runner,
						Kind:   evalKey.Kind,
						ID:     node.blockID,
						Depth:  -2, // FIXME: can be computed in the node and passed here
					},
					node.block.Meta(),
				)
				section := node.block.(*Section)

				resultsMtx.Lock()

				if section.titleToRender != nil {
					for _, n := range runQueue {
						if n.block.ID() == section.titleToRender.ID() &&
							!n.isStart { // Section start nodes do not output content
							output := results[n.id]
							if output == nil {
								_log.WarnContext(ctx, "Output of title node not found", "title_node", n.id)
								continue
							}
							contentSection.Title = output
							break
						}
					}
				}

				// collect content from all children, in order defined in the section
				for _, child := range section.childrenToRender {
					for _, n := range runQueue {
						if n.block.ID() == child.ID() && !n.isStart { // Section start nodes do not output content
							output := results[n.id]
							if output == nil {
								_log.WarnContext(ctx, "Output of child node not found", "child_node", n.id)
								continue
							}
							contentSection.Add(output)
						}
					}
				}
				resultsMtx.Unlock()

				contentSection.Compact()

				_log.DebugContext(ctx, "Storing result of section node render")
				resultsMtx.Lock()
				results[node.id] = contentSection
				resultsMtx.Unlock()

				// Section end node only collects outputs from all children
				return node, nil
			}

			// Collect outputs of the dependencies for the data context
			dependencyOutputs := plugindata.Map{}
			resultsMtx.Lock()
			for _, dep := range node.dependencies {
				parentDeps := collectBranchDependencies(ctx, _log, dep)

				for _, parentDep := range parentDeps {
					output := results[parentDep.id]
					if output == nil {
						log.WarnContext(
							ctx, "Output of the parent dependency not found",
							"parent_dependency", parentDep.id,
						)
						continue
					}
					// Note, that if multiple blocks have the same eval key, as is possible
					// with dynamic blocks, the output will be overwritten.
					dependencyOutputs.SetWithPath(parentDep.block.EvalKey().AsPath(), output.AsData())
				}
			}
			resultsMtx.Unlock()

			// PREPARE DATA CONTEXT FOR RENDERING
			predefinedDataCtx := node.block.GetDataCtx()
			var dataCtx plugindata.Map
			if predefinedDataCtx == nil {
				_log.ErrorContext(ctx, "Node received `nil` as data context")
				dataCtx = plugindata.Map{}
			} else {
				dataCtx = predefinedDataCtx.Clone()
			}
			dataCtx[plugin.DependenciesDataKey] = dependencyOutputs

			// RENDER CONTENT FOR THE NODE
			output, diags := node.block.RenderContent(ctx, dataCtx)
			if diags.HasErrors() {
				_log.ErrorContext(
					ctx, "Error while rendering content",
					"err", diagnostics.GetDiagsDetails(diags),
				)
				return nil, errors.New(diags.Error())
			}

			resultsMtx.Lock()
			results[node.id] = output
			resultsMtx.Unlock()

			return node, nil
		},
	)

	log.InfoContext(
		ctx, "Finished content rendering",
		"results_count", len(results),
		"processed_blocks", len(processedNodes),
		"errors_count", len(errs),
	)

	if len(errs) > 0 {
		var diags diagnostics.Diag
		var errsJoined error
		for _, e := range errs {
			log.ErrorContext(ctx, "Error while rendering content", "err", e)
			if d, ok := e.(diagnostics.Diag); ok {
				diags.Extend(d)
			} else {
				errsJoined = errors.Join(errsJoined, e)
			}
		}
		if errsJoined != nil {
			diags.Extend(diagnostics.FromErr(errsJoined))
		}

		if len(diags) > 0 {
			return nil, diags
		}
		return nil, errors.New("errors while rendering content")
	}
	return results, nil
}

func executeContentBlocksAsync(
	ctx context.Context,
	doc *Document,
	requiredTags []string,
	dataCtx plugindata.Map,
) (rootSection *plugin.ContentSection, diags diagnostics.Diag) {
	log := appctx.Log(ctx)

	depth := 0
	branches := []RenderableContent{}

	var titleBranch RenderableContent
	if doc.Title != nil {
		log.InfoContext(ctx, "Evaluating title block")
		branch, diag := evaluateContentTree(ctx, requiredTags, doc.Title, depth, &dataCtx)
		if diags.Extend(diag) {
			return nil, diags
		}
		if branch != nil {
			branches = append(branches, branch)
			titleBranch = branch
		}
	}

	log.InfoContext(ctx, "Evaluating content blocks")
	for _, block := range doc.ContentTreeBlocks {
		branch, diag := evaluateContentTree(ctx, requiredTags, block, depth, &dataCtx)
		if diags.Extend(diag) {
			return nil, diags
		}
		if branch != nil {
			branches = append(branches, branch)
		}
	}

	log.InfoContext(ctx, "Content tree evaluated", "branches_count", len(branches))

	nodes, err := fillInDependsOnRefs(ctx, branches)
	if err != nil {
		return nil, diagnostics.FromErr(err)
	}

	outputs, err := renderContentAsync(ctx, nodes)
	if err != nil {
		return nil, diagnostics.FromErr(err)
	}

	rootSection = plugin.NewEmptySection(
		&plugin.BlockDetails{
			Kind:  definitions.BlockKindDocument,
			Name:  doc.GetTemplateName(),
			ID:    "document",
			Depth: 0,
		},
		doc.Meta(),
	)

	if titleBranch != nil {
		for _, node := range nodes {
			if node.block == titleBranch && !node.isStart { // Section start nodes do not output content
				output := outputs[node.id]
				if output == nil {
					log.WarnContext(ctx, "Output of the title node not found", "node", node.id)
					break
				}
				rootSection.Title = output
			}
		}
	}

	for _, branch := range branches {
		for _, node := range nodes {
			if node.block != titleBranch && node.block == branch &&
				!node.isStart { // Section start nodes do not output content
				output := outputs[node.id]
				if output == nil {
					log.WarnContext(ctx, "Output of the node not found", "node", node.id)
					break
				}
				rootSection.Add(output)
				// log.WarnContext(ctx, "ADDING OUTPUT", "output", output)
			}
		}
	}

	return rootSection, diags
}

func evalIsIncludedAttr(
	ctx context.Context,
	isIncludedAttr *dataspec.Attr,
	dataCtx plugindata.Map,
) (val bool, diags diagnostics.Diag) {
	isIncluded, diag := dataspec.EvalAttr(ctx, isIncludedAttr, dataCtx)
	if diags.Extend(diag) {
		return false, diags
	}
	if isIncluded.IsNull() {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid attribute",
			Detail:   "Attribute value is unknown",
			Subject:  &isIncludedAttr.ValueRange,
		})
		return false, diags
	}

	rawVal := plugindata.Encapsulated.MustFromCty(isIncluded)
	if rawVal == nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid attribute value",
			Detail:   "Can't evaluate an attribute value",
			Subject:  &isIncludedAttr.ValueRange,
		})
		return false, diags
	}

	isTrue, err := plugindata.IsTruthy(*rawVal)
	if err != nil {
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid attribute value",
			Detail:   fmt.Sprintf("Error while evaluating the attribute value: %s", err),
			Subject:  &isIncludedAttr.ValueRange,
		})
		return false, diags
	}

	return isTrue, diags
}

func applyBlockDataToDataCtx(
	ctx context.Context,
	id string,
	name string,
	kind string,
	vars *definitions.Vars,
	requiredVars []string,
	meta plugindata.Map,
	runnerName string,
	depth int,
	dataCtx plugindata.Map,
) (diags diagnostics.Diag) {
	diag := ApplyVars(ctx, vars, dataCtx)
	if diags.Extend(diag) {
		return diags
	}
	if len(requiredVars) > 0 {
		diag = verifyRequiredVars(dataCtx, requiredVars)
		diags.Extend(diag)
	}

	dataCtx[definitions.BlockKindMeta] = meta

	// Setting details about the block itself
	details := plugin.BlockDetails{
		Kind:   kind,
		Runner: runnerName,
		Name:   name,
		ID:     id,
		Depth:  depth,
	}
	dataCtx[plugin.BlockDetailsDataKey] = details.AsData()
	return diags
}

func evaluateContentTree(
	ctx context.Context,
	requiredTags []string,
	block ContentTreeEvalBlock,
	depth int,
	dataCtx *plugindata.Map,
) (_ RenderableContent, diags diagnostics.Diag) {
	log := appctx.Log(ctx)
	log = log.With("block", block.EvalKey().AsName())
	// log.DebugContext(ctx, "Evaluating content tree block", "kind", block.Kind())

	switch block.(type) {
	case *PluginContentAction:
		contentBlock := block.(*PluginContentAction)

		if !contentBlock.meta.MatchesTags(requiredTags) {
			return nil, nil
		}

		// Update the context with block vars
		blockDataCtx := dataCtx.Clone()

		diag := applyBlockDataToDataCtx(
			ctx,
			contentBlock.ID(),
			contentBlock.EvalKey().Name,
			contentBlock.Kind(),
			contentBlock.vars,
			contentBlock.requiredVars,
			contentBlock.Meta(),
			contentBlock.RunnerName,
			depth,
			blockDataCtx,
		)
		if diags.Extend(diag) {
			// diag[0].Subject = contentBlock.Source.GetSource().Block.Range().Ptr()
			return nil, diags
		}

		isIncluded, diag := evalIsIncludedAttr(ctx, contentBlock.isIncluded, blockDataCtx)
		if diags.Extend(diag) {
			return nil, diags
		}

		if !isIncluded {
			log.DebugContext(ctx, "Skipping the block as it's not included")
			return nil, diags
		}

		contentBlock.dataCtx = &blockDataCtx
		return contentBlock, diags

	case *Section:
		section := block.(*Section)

		if !section.meta.MatchesTags(requiredTags) {
			return nil, nil
		}

		// Update the context with section data
		secDataCtx := dataCtx.Clone()

		diag := applyBlockDataToDataCtx(
			ctx,
			section.ID(),
			section.EvalKey().Name,
			section.Kind(),
			section.vars,
			section.requiredVars,
			section.Meta(),
			"",
			depth,
			secDataCtx,
		)
		if diags.Extend(diag) {
			return nil, diags
		}

		isIncluded, diag := evalIsIncludedAttr(ctx, section.isIncluded, secDataCtx)
		if diags.Extend(diag) {
			return nil, diags
		}

		if !isIncluded {
			log.DebugContext(ctx, "Skipping the section as it's not included")
			return nil, diags
		}

		// Walk through children and unpack recursively
		children := []RenderableContent{}

		for _, child := range section.children {

			newChild, diag := evaluateContentTree(ctx, requiredTags, child, depth+1, &secDataCtx)
			if diags.Extend(diag) {
				continue
			}
			if newChild != nil {
				children = append(children, newChild)
			}
		}

		if section.title != nil {
			// keep the same depth for the section's title
			titleChild, diag := evaluateContentTree(ctx, requiredTags, section.title, depth+1, &secDataCtx)
			if !diags.Extend(diag) {
				section.titleToRender = titleChild
			}
		}

		section.dataCtx = &secDataCtx
		section.childrenToRender = children
		return section, diags
	case *Dynamic:
		dynamic := block.(*Dynamic)

		itemsPtr, diag := evalItemsAttr(ctx, dynamic.items, dataCtx)

		if diags.Extend(diag) || itemsPtr == nil {
			return nil, diags
		}
		items := *itemsPtr

		dynDataCtx := dataCtx.Clone()
		dynDataCtx["items"] = items

		nonDynamicChildren, diag := evaluateDynamicBlock(ctx, requiredTags, dynamic, items, depth, &dynDataCtx)
		if diags.Extend(diag) {
			return nil, diags
		}

		newName := "dynamic." + dynamic.source.BlockName

		section := &Section{
			source: &definitions.Section{
				BlockName: newName,
			},
			blockName:        newName,
			dependsOn:        dynamic.dependsOn,
			dataCtx:          &dynDataCtx,
			childrenToRender: nonDynamicChildren,
		}
		return section, diags
	default:
		diags.Add(
			"Unknown type of a content tree block",
			fmt.Sprintf(
				"Block `%s` type `%T` encountered while evaluating a content tree",
				block.EvalKey().AsName(),
				block,
			),
		)
		return nil, diags
	}
}

func collectBranchDependencies(ctx context.Context, log *slog.Logger, parent *ExecNode) []*ExecNode {
	if parent.block.Kind() != definitions.BlockKindSection && !parent.isStart {
		// both content blocks and section end nodes output content that can be used
		results := []*ExecNode{parent}
		return results
	}

	// Section start nodes do not output content but point to the section's dependencies
	parentDeps := []*ExecNode{}
	for _, grandparent := range parent.dependencies {
		grandparentDeps := collectBranchDependencies(ctx, log, grandparent)
		parentDeps = append(parentDeps, grandparentDeps...)
	}

	return parentDeps
}

func postExecProcessContentBlocks(ctx context.Context, root *plugin.ContentSection) (*plugin.ContentSection, error) {
	log := appctx.Log(ctx)

	// find and fill in TOC nodes
	root, err := builtin.FillInTOCNodes(ctx, log, root)
	if err != nil {
		return nil, err
	}

	return root, nil
}
