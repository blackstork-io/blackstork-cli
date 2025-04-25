package microsoft_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/microsoft"
	client_mocks "github.com/blackstork-io/fabric/mocks/internalpkg/microsoft"
	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/plugindata"
	"github.com/blackstork-io/fabric/plugin/plugintest"
)

type AzureOpenAITextContentTestSuite struct {
	suite.Suite
	plugin *plugin.Schema
	schema *plugin.ContentProvider
	cli    *client_mocks.AzureOpenAIClient
}

func TestAzureOpenAITextContentSuite(t *testing.T) {
	suite.Run(t, &AzureOpenAITextContentTestSuite{})
}

func (s *AzureOpenAITextContentTestSuite) SetupSuite() {
	s.plugin = microsoft.Plugin(
		"1.0.0",
		nil,
		(func(apiKey, endPoint string) (cli microsoft.AzureOpenAIClient, err error) {
			return s.cli, nil
		}),
		nil,
		nil,
	)
	s.schema = s.plugin.ContentProviders["azure_openai_text"]
}

func (s *AzureOpenAITextContentTestSuite) SetupTest() {
	s.cli = &client_mocks.AzureOpenAIClient{}
}

func (s *AzureOpenAITextContentTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *AzureOpenAITextContentTestSuite) TestSchema() {
	s.Require().NotNil(s.plugin)
	s.Require().NotNil(s.schema)
	s.NotNil(s.schema.Args)
	s.NotNil(s.schema.ContentFunc)
	s.NotNil(s.schema.Config)
}

func (s *AzureOpenAITextContentTestSuite) TestBasic() {
	s.cli.On("GetCompletions", mock.Anything, azopenai.CompletionsOptions{
		DeploymentName: to.Ptr("test"),
		MaxTokens:      to.Ptr(int32(1000)),
		Temperature:    to.Ptr(float32(0)),
		Prompt:         []string{"Tell me a story"},
		TopP:           nil,
		N:              to.Ptr(int32(1)),
	}, mock.Anything).Return(azopenai.GetCompletionsResponse{
		Completions: azopenai.Completions{
			Choices: []azopenai.Choice{
				{Text: to.Ptr("Once upon a time.")},
			},
		},
	}, nil)
	ctx := context.Background()
	dataCtx := plugindata.Map{}

	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Config: plugintest.NewTestDecoder(s.T(), s.schema.Config).
			SetAttr("api_key", cty.StringVal("testtoken")).
			SetAttr("resource_endpoint", cty.StringVal("http://test")).
			SetAttr("deployment_name", cty.StringVal("test")).
			SetAttr("api_version", cty.StringVal("2024-02-01")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("prompt", cty.StringVal("Tell me a story")).
			SetAttr("max_tokens", cty.NumberIntVal(1000)).
			SetAttr("temperature", cty.NumberFloatVal(0)).
			SetAttr("completions_count", cty.NumberIntVal(1)).
			Decode(),
		DataContext: dataCtx,
	})
	fmt.Println(diags)
	s.Nil(diags)
	//s.Equal("Once upon a time.", mdprint.PrintString(result.Content))
	s.Equal("Once upon a time.", result.Content)
}

func (s *AzureOpenAITextContentTestSuite) TestAdvanced() {
	s.cli.On("GetCompletions", mock.Anything, azopenai.CompletionsOptions{
		DeploymentName: to.Ptr("test"),
		MaxTokens:      to.Ptr(int32(1000)),
		Temperature:    to.Ptr(float32(0)),
		Prompt:         []string{"Tell me a story about BAR. {\"foo\":\"bar\"}"},
		TopP:           nil,
		N:              to.Ptr(int32(1)),
	}, mock.Anything).Return(azopenai.GetCompletionsResponse{
		Completions: azopenai.Completions{
			Choices: []azopenai.Choice{
				{Text: to.Ptr("Once upon a time.")},
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
		Config: plugintest.NewTestDecoder(s.T(), s.schema.Config).
			SetAttr("api_key", cty.StringVal("testtoken")).
			SetAttr("resource_endpoint", cty.StringVal("http://test")).
			SetAttr("deployment_name", cty.StringVal("test")).
			SetAttr("api_version", cty.StringVal("2024-02-01")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("prompt", cty.StringVal("Tell me a story about {{.local.foo | upper}}. {{ .local | toRawJson }}")).
			SetAttr("max_tokens", cty.NumberIntVal(1000)).
			SetAttr("temperature", cty.NumberFloatVal(0)).
			SetAttr("completions_count", cty.NumberIntVal(1)).
			Decode(),
		DataContext: dataCtx,
	})
	s.Empty(diags)
	//s.Equal("Once upon a time.", mdprint.PrintString(result.Content))
	s.Equal("Once upon a time.", result.Content)
}

func (s *AzureOpenAITextContentTestSuite) TestMissingPrompt() {
	plugintest.DecodeAndAssert(s.T(), s.schema.Args, "", plugindata.Map{}, diagtest.Asserts{
		{
			diagtest.IsError,
			diagtest.SummaryEquals("Missing required attribute"),
			diagtest.DetailContains("prompt"),
		},
	})
}

func (s *AzureOpenAITextContentTestSuite) TestMissingAPIKey() {
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
		{
			diagtest.IsError,
			diagtest.SummaryEquals("Missing required attribute"),
			diagtest.DetailContains("resource_endpoint"),
		},
		{
			diagtest.IsError,
			diagtest.SummaryEquals("Missing required attribute"),
			diagtest.DetailContains("deployment_name"),
		},
	})
}

func (s *AzureOpenAITextContentTestSuite) TestFailingClient() {
	s.cli.On("GetCompletions", mock.Anything, azopenai.CompletionsOptions{
		DeploymentName: to.Ptr("test"),
		MaxTokens:      to.Ptr(int32(1000)),
		Temperature:    to.Ptr(float32(0)),
		Prompt:         []string{"Tell me a story"},
		TopP:           nil,
		N:              to.Ptr(int32(1)),
	}, mock.Anything).Return(azopenai.GetCompletionsResponse{}, errors.New("failed to generate text from model"))
	ctx := context.Background()
	dataCtx := plugindata.Map{}
	result, diags := s.schema.ContentFunc(ctx, &plugin.ProvideContentParams{
		Config: plugintest.NewTestDecoder(s.T(), s.schema.Config).
			SetAttr("api_key", cty.StringVal("testtoken")).
			SetAttr("resource_endpoint", cty.StringVal("http://test")).
			SetAttr("deployment_name", cty.StringVal("test")).
			SetAttr("api_version", cty.StringVal("2024-02-01")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.schema.Args).
			SetAttr("prompt", cty.StringVal("Tell me a story")).
			SetAttr("max_tokens", cty.NumberIntVal(1000)).
			SetAttr("temperature", cty.NumberFloatVal(0)).
			SetAttr("completions_count", cty.NumberIntVal(1)).
			Decode(),
		DataContext: dataCtx,
	})
	s.Nil(result)
	s.Equal(diagnostics.Diag{{
		Severity: hcl.DiagError,
		Summary:  "Failed to generate text",
		Detail:   "failed to generate text from model",
	}}, diags)
}
