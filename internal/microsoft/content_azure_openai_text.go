package microsoft

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strings"
	"text/template"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Masterminds/sprig/v3"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/dataspec"
	"github.com/blackstork-io/fabric/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

func makeAzureOpenAITextContentSchema(loader AzureOpenAIClientLoadFn) *plugin.ContentProvider {
	return &plugin.ContentProvider{
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "api_key",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
					Secret:      true,
				},
				{
					Name:        "resource_endpoint",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
				{
					Name:        "deployment_name",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
				},
				{
					Name:       "api_version",
					Type:       cty.String,
					DefaultVal: cty.StringVal("2024-02-01"),
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
					Name:       "max_tokens",
					Type:       cty.Number,
					DefaultVal: cty.NumberIntVal(1000),
				},
				{
					Name:       "temperature",
					Type:       cty.Number,
					DefaultVal: cty.NumberFloatVal(0),
				},
				{
					Name: "top_p",
					Type: cty.Number,
				},
				{
					Name:       "completions_count",
					Type:       cty.Number,
					DefaultVal: cty.NumberIntVal(1),
				},
			},
		},
		ContentFunc: genOpenAIText(loader),
	}
}

func genOpenAIText(clientLoader AzureOpenAIClientLoadFn) plugin.ProvideContentFunc {
	return func(ctx context.Context, params *plugin.ProvideContentParams) (*plugin.ContentResult, diagnostics.Diag) {
		apiKey := params.Config.GetAttrVal("api_key").AsString()
		resourceEndpoint := params.Config.GetAttrVal("resource_endpoint").AsString()
		client, err := clientLoader(apiKey, resourceEndpoint)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to create client",
				Detail:   err.Error(),
			}}
		}
		result, err := renderText(ctx, client, params.Config, params.Args, params.DataContext)
		if err != nil {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to generate text",
				Detail:   err.Error(),
			}}
		}
		return &plugin.ContentResult{
			Content: plugin.NewElementFromMarkdown(result),
		}, nil
	}
}

func renderText(ctx context.Context, cli AzureOpenAIClient, cfg, args *dataspec.Block, dataCtx plugindata.Map) (string, error) {
	params := azopenai.CompletionsOptions{}
	params.DeploymentName = to.Ptr(cfg.GetAttrVal("deployment_name").AsString())

	maxTokens, _ := args.GetAttrVal("max_tokens").AsBigFloat().Int64()
	if maxTokens > 0 {
		if maxTokens > math.MaxInt32 {
			maxTokens = math.MaxInt32
		}
		params.MaxTokens = to.Ptr(int32(maxTokens)) //nolint:gosec // G115: safe conversion, range checked
	}

	temperature, _ := args.GetAttrVal("temperature").AsBigFloat().Float32()
	params.Temperature = to.Ptr(temperature)

	completionsCount, _ := args.GetAttrVal("completions_count").AsBigFloat().Int64()
	if completionsCount > 0 {
		if completionsCount > math.MaxInt32 {
			completionsCount = math.MaxInt32
		}
		params.N = to.Ptr(int32(completionsCount)) //nolint:gosec // G115: safe conversion, range checked
	}

	topPAttr := args.GetAttrVal("top_p")
	if !topPAttr.IsNull() {
		topP, _ := topPAttr.AsBigFloat().Float32()
		params.TopP = to.Ptr(topP)
	}

	renderedPrompt, err := templateText(args.GetAttrVal("prompt").AsString(), dataCtx)
	if err != nil {
		return "", err
	}
	params.Prompt = []string{renderedPrompt}
	// TODO: use api version from config
	resp, err := cli.GetCompletions(ctx, params, nil)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", nil
	}
	return *resp.Choices[0].Text, nil
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
