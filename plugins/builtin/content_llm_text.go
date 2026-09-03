// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package builtin

import (
	"context"
	"errors"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/anthropic"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/ollama"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec/constraint"
)

// Client abstracts LLM generation operations utilizing Google's Genkit.
type Client interface {
	Init(ctx context.Context, vendor string)
	GenerateText(ctx context.Context, modelName, systemPrompt, prompt string) (string, error)
}

type genkitClient struct {
	apiKey string
	g      *genkit.Genkit
}

type Option func(*genkitClient)

// WithAPIKey records the API key on the client.
func WithAPIKey(apiKey string) Option {
	return func(c *genkitClient) {
		c.apiKey = apiKey
	}
}

// NewClient creates a new Genkit-backed LLM wrapper client.
func NewClient(opts ...Option) Client {
	c := &genkitClient{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *genkitClient) Init(ctx context.Context, vendor string) {
	switch vendor {
	case "google":
		c.g = genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{
			APIKey: c.apiKey,
		}))
	case "openai":
		c.g = genkit.Init(ctx, genkit.WithPlugins(&openai.OpenAI{
			APIKey: c.apiKey,
		}))
	case "anthropic":
		c.g = genkit.Init(ctx, genkit.WithPlugins(&anthropic.Anthropic{
			APIKey: c.apiKey,
		}))
	case "ollama":
		c.g = genkit.Init(ctx, genkit.WithPlugins(
			&ollama.Ollama{
				ServerAddress: "http://localhost:11434",
			},
		))
	case "xai":
		c.g = genkit.Init(
			ctx,
			genkit.WithPlugins(&compat_oai.OpenAICompatible{
				Provider: "xai",
				APIKey:   c.apiKey,
				BaseURL:  "https://api.x.ai/v1",
			}),
		)
	}
}

// GenerateText passes a fully-formed chat request through the Genkit abstraction layer.
func (c *genkitClient) GenerateText(ctx context.Context, modelName, systemPrompt, prompt string) (string, error) {
	if modelName == "" {
		return "", errors.New("model name is required")
	}
	if prompt == "" {
		return "", errors.New("prompt is required")
	}

	var messages []*ai.Message

	if systemPrompt != "" {
		messages = append(messages, &ai.Message{
			Role:    ai.RoleSystem,
			Content: []*ai.Part{ai.NewTextPart(systemPrompt)},
		})
	}

	messages = append(messages, &ai.Message{
		Role:    ai.RoleUser,
		Content: []*ai.Part{ai.NewTextPart(prompt)},
	})

	resp, err := genkit.Generate(
		ctx, c.g,
		ai.WithModelName(modelName),
		ai.WithMessages(messages...),
	)
	if err != nil {
		return "", fmt.Errorf("text generation failed: %w", err)
	}

	return resp.Text(), nil
}

func makeLLMTextContentProvider() *plugin.ContentProvider {
	return &plugin.ContentProvider{
		Doc: "Generates text from a Go-templated prompt using the configured LLM vendor and model. Supports Google, OpenAI, Anthropic, Ollama, and xAI models.",
		Config: &dataspec.RootSpec{
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "vendor",
					Type:        cty.String,
					Doc:         `LLM vendor name`,
					Constraints: constraint.RequiredNonNull,
					OneOf: []cty.Value{
						cty.StringVal("google"),
						cty.StringVal("openai"),
						cty.StringVal("anthropic"),
						cty.StringVal("ollama"),
						cty.StringVal("xai"),
					},
					ExampleVal: cty.StringVal("google"),
				},
				{
					Name:        "model",
					Type:        cty.String,
					Doc:         `Model name`,
					Constraints: constraint.RequiredNonNull,
					ExampleVal:  cty.StringVal("googleai/gemini-3-flash"),
				},
				{
					Name:        "api_key",
					Type:        cty.String,
					Constraints: constraint.RequiredNonNull,
					ExampleVal:  cty.StringVal("key_value"),
					Secret:      true,
				},
				{
					Name: "system_prompt",
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
			},
		},
		ContentFunc: genLLMContent(),
	}
}

func genLLMContent() plugin.ProvideContentFunc {
	return func(ctx context.Context, params *plugin.ProvideContentParams) (*plugin.ContentProviderResult, error) {
		var opts []Option

		apiKey := params.Config.GetAttrVal("api_key")
		if !apiKey.IsNull() && apiKey.AsString() != "" {
			opts = append(opts, WithAPIKey(apiKey.AsString()))
		}

		vendor := params.Config.GetAttrVal("vendor").AsString()

		client := NewClient(opts...)
		client.Init(ctx, vendor)

		modelName := params.Config.GetAttrVal("model").AsString()

		systemPrompt := ""
		sysPromptAttr := params.Config.GetAttrVal("system_prompt")
		if !sysPromptAttr.IsNull() && sysPromptAttr.AsString() != "" {
			systemPrompt = sysPromptAttr.AsString()
		}

		promptTmpl := params.Args.GetAttrVal("prompt").AsString()

		prompt, err := renderText(promptTmpl, params.DataContext)
		if err != nil {
			return nil, err
		}

		output, err := client.GenerateText(ctx, modelName, systemPrompt, prompt)
		if err != nil {
			return nil, fmt.Errorf("failed to generate content: %w", err)
		}

		return &plugin.ContentProviderResult{
			Content: plugin.NewTextElement(output),
		}, nil
	}
}
