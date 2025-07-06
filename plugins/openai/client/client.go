package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type client struct {
	baseURL string
	apiKey  string
	orgID   string
}

var defaultClient = client{
	baseURL: "https://api.openai.com",
}

// New creates a new OpenAI API client
func New(o ...Option) Client {
	c := defaultClient
	for _, opt := range o {
		opt(&c)
	}
	return &c
}

// Client implements the OpenAI API client with minimal functionality
type Client interface {
	GenerateChatCompletion(ctx context.Context, params *ChatCompletionParams) (*ChatCompletionResult, error)
}

func (c *client) auth(r *http.Request) {
	r.Header.Set("Authorization", "Bearer "+c.apiKey)
	if c.orgID != "" {
		r.Header.Set("OpenAI-Organization", c.orgID)
	}
}

func (c *client) contentType(r *http.Request) {
	r.Header.Set("Content-Type", "application/json")
}

// GenerateChatCompletion generates a chat completion
func (c *client) GenerateChatCompletion(ctx context.Context, params *ChatCompletionParams) (*ChatCompletionResult, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.auth(req)
	c.contentType(req)
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		var errRes ErrorResponse
		if err := json.NewDecoder(res.Body).Decode(&errRes); err != nil {
			return nil, fmt.Errorf("failed to decode error response: %w", err)
		}
		return nil, errRes.Error
	}
	var result ChatCompletionResult
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
