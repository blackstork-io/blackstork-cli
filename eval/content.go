package eval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/blackstork-io/fabric/parser/definitions"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

// Identifies a content tree block
type EvalKey struct {
	kind   string
	runner string
	name   string
}

func (key EvalKey) AsName() string {
	name := key.kind
	if key.runner != "" {
		name += "." + key.runner
	}
	name += "." + key.name
	return name
}

func (key EvalKey) AsPath() []string {
	path := []string{key.kind}
	if key.runner != "" {
		path = append(path, key.runner)
	}
	path = append(path, key.name)
	return path
}

func EvalKeyFromDefKey(key definitions.Key) EvalKey {
	return EvalKey{
		kind:   key.Kind,
		runner: key.Runner,
		name:   key.Name,
	}
}

type ContentTreeEvalBlock interface {
	isContentTreeEvalBlock()

	EvalKey() EvalKey
	Kind() string

	addNameSuffix(val string)
}

type RenderableContent interface {
	RenderContent(ctx context.Context, data plugindata.Map) (plugin.Content, diagnostics.Diag)
	EvalKey() EvalKey
	Kind() string

	Meta() plugindata.Map

	GetDataCtx() *plugindata.Map
}


func LoadContent(
	ctx context.Context,
	providers ContentProviders,
	block definitions.ContentTreeBlock,
) (_ ContentTreeEvalBlock, diags diagnostics.Diag) {
	var treeBlock ContentTreeEvalBlock
	switch block := block.(type) {
	case *definitions.ContentBlock:
		treeBlock, diags = LoadPluginContentAction(ctx, providers, block)
	case *definitions.Section:
		treeBlock, diags = LoadSection(ctx, providers, block)
	case *definitions.Dynamic:
		treeBlock, diags = LoadDynamic(ctx, providers, block)
	default:
		diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Error while loading content: unsupported content tree block type: %T", block),
			Detail:   "Content tree block must be either `content`, `section` or `dynamic`",
		})
	}
	if diags.HasErrors() {
		return nil, diags
	}
	return treeBlock, diags
}
