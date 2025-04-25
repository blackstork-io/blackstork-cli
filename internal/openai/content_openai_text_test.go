package openai

import (
	"context"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/blackstork-io/fabric/internal/openai/client"
	client_mocks "github.com/blackstork-io/fabric/mocks/internalpkg/openai/client"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/plugindata"
	"github.com/blackstork-io/fabric/plugin/plugintest"
)

type OpenAITextContentTestSuite struct {
	suite.Suite
	schema *plugin.ContentProvider
	cli    *client_mocks.Client
}

func TestOpenAITextContentSuite(t *testing.T) {
	suite.Run(t, &OpenAITextContentTestSuite{})
}

func (s *OpenAITextContentTestSuite) SetupSuite() {
	s.schema = makeOpenAITextContentSchema(func(opts ...client.Option) client.Client {
		return s.cli
	})
}

func (s *OpenAITextContentTestSuite) SetupTest() {
	s.cli = &client_mocks.Client{}
}

func (s *OpenAITextContentTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *OpenAITextContentTestSuite) TestSchema() {
	s.Require().NotNil(s.schema)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.ContentFunc)
	s.NotNil(s.schema.Config)
}

func (s *OpenAITextContentTestSuite) TestBasic() {
	s.cli.On("GenerateChatCompletion", mock.Anything, &client.ChatCompletionParams{
		Model: defaultModel,
		Messages: []client.ChatCompletionMessage{
			{
				Role:    "user",
				Content: "Tell me a story",
			},
		},
	}).Return(&client.ChatCompletionResult{
		Choices: []client.ChatCompletionChoice{
			{
				FinishedReason: "stop",
				Index:          0,
				Message: client.ChatCompletionMessage{
					Role:    "assistant",
					Content: "Once upon a time.",
				},
			},
		},
	}, nil)
	ctx := context.Background()
	dataCtx := plugindata.Map{}
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
			prompt = "Tell me a story"
		`, dataCtx, diagtest.Asserts{}),
		Config: plugintest.DecodeAndAssert(s.T(), s.schema.Config, `
			api_key = "api_key_123"
		`, dataCtx, diagtest.Asserts{}),
		DataContext: dataCtx,
	})
	s.Nil(diags)
	s.Equal("Once upon a time.", result.Content)
}

func (s *OpenAITextContentTestSuite) TestAdvanced() {
	s.cli.On("GenerateChatCompletion", mock.Anything, &client.ChatCompletionParams{
		Model: "model_123",
		Messages: []client.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "Some system message.",
			},
			{
				Role:    "user",
				Content: "Tell me a story about BAR. {\"foo\":\"bar\"}",
			},
		},
	}).Return(&client.ChatCompletionResult{
		Choices: []client.ChatCompletionChoice{
			{
				FinishedReason: "stop",
				Index:          0,
				Message: client.ChatCompletionMessage{
					Role:    "assistant",
					Content: "Once upon a time.",
				},
			},
		},
	}, nil)
	ctx := context.Background()
	dataCtx := plugindata.Map{
		"local": plugindata.Map{
			"foo": plugindata.String("bar"),
		},
	}

	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
			prompt = "Tell me a story about {{.local.foo | upper}}. {{ .local | toRawJson }}"
			model = "model_123"
		`, dataCtx, diagtest.Asserts{}),
		Config: plugintest.DecodeAndAssert(s.T(), s.schema.Config, `
			api_key = "api_key_123"
			organization_id = "org_id_123"
			system_prompt = "Some system message."
		`, dataCtx, diagtest.Asserts{}),
		DataContext: dataCtx,
	})
	s.Empty(diags)
	s.Equal("Once upon a time.", result.Content)
}

func (s *OpenAITextContentTestSuite) TestMissingPrompt() {
	plugintest.DecodeAndAssert(s.T(), s.schema.Args, "", plugindata.Map{}, diagtest.Asserts{
		{
			diagtest.IsError,
			diagtest.SummaryEquals("Missing required attribute"),
			diagtest.DetailContains("prompt"),
		},
	})
}

func (s *OpenAITextContentTestSuite) TestMissingAPIKey() {
	plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
		prompt = "Tell me a story"
	`, plugindata.Map{}, diagtest.Asserts{})
	plugintest.DecodeAndAssert(s.T(), s.schema.Config, `
	`, plugindata.Map{}, diagtest.Asserts{
		{
			diagtest.IsError,
			diagtest.SummaryEquals("Missing required attribute"),
			diagtest.DetailContains("api_key"),
		},
	})
}

func (s *OpenAITextContentTestSuite) TestFailingClient() {
	s.cli.On("GenerateChatCompletion", mock.Anything, mock.Anything).Return(nil, client.Error{
		Type:    "error_type",
		Message: "error_message",
	})
	ctx := context.Background()
	dataCtx := plugindata.Map{}
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
			prompt = "Tell me a story"
		`, dataCtx, diagtest.Asserts{}),
		Config: plugintest.DecodeAndAssert(s.T(), s.schema.Config, `
			api_key = "api_key_123"
		`, dataCtx, diagtest.Asserts{}),
		DataContext: dataCtx,
	})
	s.Nil(result)
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to generate text",
		Detail:   "openai[error_type]: error_message",
	}}, diags)
}

func (s *OpenAITextContentTestSuite) TestCancellation() {
	s.cli.On("GenerateChatCompletion", mock.Anything, mock.Anything).Return(nil, context.Canceled)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dataCtx := plugindata.Map{}
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
			prompt = "Tell me a story"
		`, dataCtx, diagtest.Asserts{}),
		Config: plugintest.DecodeAndAssert(s.T(), s.schema.Config, `
			api_key = "api_key_123"
		`, dataCtx, diagtest.Asserts{}),
		DataContext: dataCtx,
	})
	s.Nil(result)
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to generate text",
		Detail:   "context canceled",
	}}, diags)
}

func (s *OpenAITextContentTestSuite) TestErrorEncoding() {
	want := client.Error{
		Type:    "invalid_request_error",
		Message: "message of error",
	}
	s.cli.On("GenerateChatCompletion", mock.Anything, mock.Anything).Return(nil, want)
	ctx := context.Background()
	dataCtx := plugindata.Map{}
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Args: plugintest.DecodeAndAssert(s.T(), s.schema.Args, `
			prompt = "Tell me a story"
		`, dataCtx, diagtest.Asserts{}),
		Config: plugintest.DecodeAndAssert(s.T(), s.schema.Config, `
			api_key = "api_key_123"
		`, dataCtx, diagtest.Asserts{}),
		DataContext: dataCtx,
	})
	s.Nil(result)
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to generate text",
		Detail:   "openai[invalid_request_error]: message of error",
	}}, diags)
}
