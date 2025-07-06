package eval

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/blackstork-io/fabric/cmd/fabctx"
	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/utils"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
	"github.com/google/uuid"
	"github.com/hashicorp/hcl/v2"
)

const (
	maxWorkerCount = 5

	outputDataContextKey = "output"
)

type ExecNode struct {
	id string

	evalKey EvalKey

	block  RenderableContent
	parent RenderableContent

	isStart bool
	isEnd   bool

	dependencies []*ExecNode
	dependants   []*ExecNode

	wg *sync.WaitGroup
}

func makeID(key EvalKey, parent RenderableContent, isStart, isEnd bool) string {

	name := key.AsName()

	if parent != nil {
		name = fmt.Sprintf("%s-%s", parent.EvalKey().AsName(), name)
	}

	// To make sure the ID is unique
	uniquePrefix := uuid.New().String()[:8]

	prefixedName := fmt.Sprintf("%s-%s", uniquePrefix, name)

	if isStart {
		prefixedName += ":start"
	}
	if isEnd {
		prefixedName += ":end"
	}
	return prefixedName
}

func getDependencies(block RenderableContent) ([]EvalKey, error) {
	dependencies := []EvalKey{}

	// If there is a parent, depend on it - the parent might contain
	// dependencies, so it can be used as a group dependency lock.
	// if n.parentSection != nil {
	// 	dependencies = append(dependencies, n.parentSection.block.EvalKey())
	// }

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
) (res []*ExecNode) {
	key := block.EvalKey()

	if block.Kind() == definitions.BlockKindSection {
		// splitting every section node into 2 nodes: start and end nodes
		startNode := &ExecNode{
			id:      makeID(key, parent, true, false),
			evalKey: key,
			block:   block,
			parent:  parent,
			isStart: true,
		}
		endNode := &ExecNode{
			id:      makeID(key, parent, false, true),
			evalKey: key,
			block:   block,
			parent:  parent,
			isEnd:   true,
		}
		res = append(res, startNode, endNode)

		section := block.(*Section)
		children := section.childrenToRender

		for _, child := range children {
			childNodes := createExecNodesRecursive(ctx, block, child)
			res = append(res, childNodes...)
		}

	} else {
		node := &ExecNode{
			id:      makeID(key, parent, false, false),
			evalKey: key,
			block:   block,
			parent:  parent,
		}
		res = append(res, node)
	}
	return res
}

func wireNodeDependencies(ctx context.Context, nodes []*ExecNode) ([]*ExecNode, error) {

	log := fabctx.GetLog(ctx)

	for _, node := range nodes {

		if node.block.Kind() == definitions.BlockKindSection && node.isEnd {
			// Section end nodes only depend on the children of the section and are wired after all
			// other blocks are hooked in
			continue
		}

		dependencies, err := getDependencies(node.block)
		if err != nil {
			log.ErrorContext(
				ctx, "Error getting dependencies for a block",
				"block", node.evalKey.AsName(),
				"err", err,
			)
			return nil, err
		}

		for _, depKey := range dependencies {
			var depNode *ExecNode

			for _, n := range nodes {
				if n.evalKey != depKey {
					continue
				}

				// if the dependency is a section, depend on its end block
				if depKey.kind == definitions.BlockKindSection && n.isEnd {
					depNode = n
					break
				}

				// ignore section start nodes in this pass
				if depKey.kind == definitions.BlockKindSection && n.isStart {
					continue
				}

				depNode = n
			}

			if depNode == nil {
				log.ErrorContext(
					ctx, "Dependency not found in the template",
					"dependency", depKey.AsName(),
					"block", node.evalKey.AsName(),
				)
				return nil, fmt.Errorf("dependency `%s` not found", depKey.AsName())
			}

			node.dependencies = append(node.dependencies, depNode)
			depNode.dependants = append(depNode.dependants, node)
		}

		// if node has a parent, the node must depend on the parent's start node
		if node.parent != nil {
			parentKey := node.parent.EvalKey()

			var parentStartNode *ExecNode

			for _, n := range nodes {
				if n.evalKey == parentKey && n.isStart {
					parentStartNode = n
					break
				}
			}

			if parentStartNode == nil {
				log.ErrorContext(
					ctx, "Parent start node not found in the node list",
					"parent", parentKey.AsName(),
					"block", node.evalKey.AsName(),
				)
				return nil, fmt.Errorf("parent start node `%s` not found", parentKey.AsName())
			}

			node.dependencies = append(node.dependencies, parentStartNode)
			parentStartNode.dependants = append(parentStartNode.dependants, node)
		}
	}

	for _, node := range nodes {
		if node.block.Kind() != definitions.BlockKindSection || !node.isEnd {
			continue
		}

		var twinStartNode *ExecNode
		for _, n := range nodes {
			if n.evalKey == node.evalKey && n.isStart {
				twinStartNode = n
				break
			}
		}
		if twinStartNode == nil {
			log.ErrorContext(
				ctx, "Sibling start node not found",
				"block", node.evalKey.AsName(),
			)
			return nil, fmt.Errorf("start node for section `%s` not found", node.evalKey.AsName())
		}

		// Depend on all blocks that wait on the start node
		for _, startDependant := range twinStartNode.dependants {

			// if one of the dependants a subsection start, find it's end
			if startDependant.isStart {
				found := false
				for _, _n := range nodes {
					if _n.evalKey == startDependant.evalKey && _n.isEnd {
						startDependant = _n
						found = true
						break
					}
				}
				if !found {
					log.ErrorContext(
						ctx, "Sibling start node for a subsection not found",
						"block", startDependant.evalKey.AsName(),
					)
					return nil, fmt.Errorf("start node for section `%s` not found", startDependant.evalKey.AsName())
				}
			}

			node.dependencies = append(node.dependencies, startDependant)
			startDependant.dependants = append(startDependant.dependants, node)
		}
	}

	return nodes, nil
}

func fillInDependsOnRefs(ctx context.Context, blocks []RenderableContent) ([]*ExecNode, error) {

	log := fabctx.GetLog(ctx)
	log.InfoContext(ctx, "Filling in the depedency references")

	nodes := []*ExecNode{}

	for _, b := range blocks {
		subnodes := createExecNodesRecursive(ctx, nil, b)
		nodes = append(nodes, subnodes...)
	}

	nodeNames := []string{}
	for _, n := range nodes {
		nodeNames = append(nodeNames, n.id)
	}

	log.DebugContext(
		ctx, "Catalog all template block nodes",
		"blocks_count", len(blocks),
		"nodes_count", len(nodes),
		"node_names", strings.Join(nodeNames, ", "),
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

	log := fabctx.GetLog(ctx)
	log.DebugContext(
		ctx, "Detecting dependency cycles",
		"nodes_count", len(nodes),
	)

	degrees := map[string]int{}
	for _, n := range nodes {
		for _, dependant := range n.dependants {
			//fmt.Printf("node: %s, dependant: %s\n", n.id, dependant.id)
			degree, ok := degrees[dependant.id]
			if !ok {
				degree = 0
			}
			degrees[dependant.id] = degree + 1
		}
	}
	// for id, degree := range degrees {
	// 	fmt.Printf("%s: %d\n", id, degree)
	// }

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
			k := n.parent.EvalKey()
			parentKey = &k
		}

		log.DebugContext(
			ctx, "Processed dependencies for a template node",
			"node", n.id,
			"parent", parentKey,
			"dependencies_count", len(n.dependencies),
			"dependants_count", len(n.dependants),
		)

		for _, dependant := range n.dependants {
			degree, ok := degrees[dependant.id]

			if !ok {
				log.ErrorContext(
					ctx, "Incorrect dependency in degree for the block",
					"node", dependant.id,
				)
				return fmt.Errorf("incorrect dependency in degree for node `%s`", dependant.id)
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

	log := fabctx.GetLog(ctx)
	log.DebugContext(
		ctx, "Rendering the content",
		"nodes_count", len(nodes),
	)

	degrees := map[string]int{}
	for _, n := range nodes {
		for _, dependant := range n.dependants {
			//fmt.Printf("node: %s, dependant: %s\n", n.id, dependant.id)
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
				"node", node.id,
				"block", node.evalKey.AsName(),
			)
			_log.DebugContext(ctx, "Rendering a template node")

			depIDs := []string{}
			for _, d := range node.dependencies {
				depIDs = append(depIDs, d.id)
			}
			_log.DebugContext(ctx, "Waiting for the node dependencies", "deps", depIDs)

			// Actual waiting for dependencies
			for _, dep := range node.dependencies {
				dep.wg.Wait()
			}

			if node.evalKey.kind == definitions.BlockKindSection && node.isStart {
				// Section start node has nothing to do, except waiting for the external dependencies
				return node, nil
			}

			if node.evalKey.kind == definitions.BlockKindSection && node.isEnd {

				contentSection := plugin.NewEmptySection(
					plugin.BlockSelf{
						Name: node.evalKey.AsName(),
					},
					node.block.Meta(),
				)
				section := node.block.(*Section)

				resultsMtx.Lock()
				// collect content from all children, in order defined in the section
				for _, child := range section.childrenToRender {
					for _, n := range runQueue {
						if n.block == child && !n.isStart { // Section start nodes do not output content
							output := results[n.id]
							if output == nil {
								_log.WarnContext(
									ctx, "Output of the child node not found",
									"child_node", n.id,
									"block", node.evalKey.AsName(),
								)
								continue
							}
							contentSection.Add(output)
						}
					}
				}
				resultsMtx.Unlock()

				contentSection.Compact()

				_log.DebugContext(ctx, "Storing the result of the section render")
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
					dependencyOutputs.SetWithPath(parentDep.evalKey.AsPath(), output.AsData())
				}
			}
			resultsMtx.Unlock()

			dataCtx := node.block.GetDataCtx()
			if dataCtx == nil {
				_log.ErrorContext(ctx, "Node received `nil` as data context")
				dataCtx = &plugindata.Map{}
			}

			(*dataCtx)[outputDataContextKey] = dependencyOutputs

			// _log.InfoContext(
			// 	ctx, "Rendering a node",
			// 	"vars", (*dataCtx)["vars"],
			// )

			output, diags := node.block.RenderContent(ctx, *dataCtx)
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
		ctx, "Finished content tree rendering",
		"results_count", len(results),
		"processed_blocks", len(processedNodes),
		"errors_count", len(errs),
	)

	if len(errs) > 0 {
		for _, e := range errs {
			log.ErrorContext(ctx, "Error while rendering a content tree", "err", e)
		}
		return nil, errors.New("multiple errors while rendering a content tree")
	}
	return results, nil
}

func executeContentBlocksAsync(
	ctx context.Context,
	doc *Document,
	requiredTags []string,
	dataCtx plugindata.Map,
) (rootSection *plugin.ContentSection, diags diagnostics.Diag) {

	log := fabctx.GetLog(ctx)

	branches := []RenderableContent{}

	log.InfoContext(ctx, "Evaluating the content blocks")
	for _, block := range doc.ContentTreeBlocks {
		branch, diag := evaluateContentTree(ctx, requiredTags, block, &dataCtx)
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

	rootSelf := plugin.BlockSelf{
		Name: doc.GetName(),
	}

	rootSection = plugin.NewEmptySection(rootSelf, doc.Meta())
	for _, branch := range branches {
		for _, node := range nodes {
			if node.block == branch && !node.isStart { // Section start nodes do not output content
				output := outputs[node.id]
				if output == nil {
					log.WarnContext(ctx, "Output of the node not found", "node", node.id)
					break
				}
				rootSection.Add(output)
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
	name string,
	vars *definitions.Vars,
	requiredVars []string,
	meta plugindata.Map,
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
	dataCtx[plugin.SelfDataCtxKey] = plugindata.Map{
		plugin.SelfNameDataCtxKey: plugindata.String(name),
	}

	return diags
}

func evaluateContentTree(
	ctx context.Context,
	requiredTags []string,
	block ContentTreeEvalBlock,
	dataCtx *plugindata.Map,
) (_ RenderableContent, diags diagnostics.Diag) {

	log := fabctx.GetLog(ctx)
	log = log.With("block", block.EvalKey().AsName())
	log.DebugContext(ctx, "Evaluating a block")

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
			contentBlock.EvalKey().AsName(),
			contentBlock.vars,
			contentBlock.requiredVars,
			contentBlock.Meta(),
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

		//section := utils.Clone(originalSection)
		// Update section vars
		// section.vars = originalSectionBlock.vars.Extend(varVals)

		// Update the context with section data
		secDataCtx := dataCtx.Clone()

		diag := applyBlockDataToDataCtx(
			ctx,
			section.EvalKey().AsName(),
			section.vars,
			section.requiredVars,
			section.Meta(),
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
			newChild, diag := evaluateContentTree(ctx, requiredTags, child, &secDataCtx)
			if diags.Extend(diag) {
				continue
			}
			if newChild != nil {
				children = append(children, newChild)
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

		nonDynamicChildren, diag := evaluateDynamicBlock(ctx, requiredTags, dynamic, items, &dynDataCtx)
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

	if parent.evalKey.kind != definitions.BlockKindSection && !parent.isStart {
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
