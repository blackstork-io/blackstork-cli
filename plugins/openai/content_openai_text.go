package openai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/openai/client"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

func makeOpenAITextContentSchema(loader ClientLoadFn) *plugin.ContentProvider {
	return &plugin.ContentProvider{
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name: "system_prompt",
					Type: cty.String,
				},
				{
					Name:        "api_key",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
					Secret:      true,
				},
				{
					Name: "organization_id",
					Type: cty.String,
				},
			},
		},
		Args: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "prompt",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
					ExampleVal:  cty.StringVal("Summarize the following text: {{.vars.text_to_summarize}}"),
				},
				{
					Name:        "model",
					Type:        cty.String,
					Constraints: constraint.Meaningful,
					DefaultVal:  cty.StringVal(defaultModel),
				},
			},
		},
		ContentFunc: genOpenAIText(loader),
	}
}

func genOpenAIText(loader ClientLoadFn) plugin.ProvideContentFunc {
	return func(ctx context.Context, params *plugin.ProvideContentParams) (*plugin.ContentProviderResult, diagnostics.Diag) {
		cli, err := makeClient(loader, params.Config)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to create client",
				Detail:   err.Error(),
			}}
		}
		result, err := renderText(ctx, cli, params.Config, params.Args, params.DataContext)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to generate text",
				Detail:   err.Error(),
			}}
		}
		return &plugin.ContentProviderResult{
			Content: plugin.NewTextElement(result),
		}, nil
	}
}

func renderText(
	ctx context.Context,
	cli client.Client,
	cfg, args *dataspec.Block,
	dataCtx plugindata.Map,
) (string, error) {
	params := client.ChatCompletionParams{
		Model: args.GetAttrVal("model").AsString(),
	}
	systemPrompt := cfg.GetAttrVal("system_prompt")
	if !systemPrompt.IsNull() && systemPrompt.AsString() != "" {
		params.Messages = append(params.Messages, client.ChatCompletionMessage{
			Role:    "system",
			Content: systemPrompt.AsString(),
		})
	}
	content, err := templateText(args.GetAttrVal("prompt").AsString(), dataCtx)
	if err != nil {
		return "", err
	}
	params.Messages = append(params.Messages, client.ChatCompletionMessage{
		Role:    "user",
		Content: content,
	})
	result, err := cli.GenerateChatCompletion(ctx, &params)
	if err != nil {
		return "", err
	}
	if len(result.Choices) < 1 {
		return "", errors.New("no choices")
	}
	return result.Choices[0].Message.Content, nil
}

func templateText(text string, dataCtx plugindata.Map) (string, error) {
	tmpl, err := template.New("text").Funcs(sprig.FuncMap()).Parse(text)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, dataCtx.Any())
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return strings.TrimSpace(buf.String()), nil
}
