package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

// ClientTestSuite is a test suite for contract testing the Client to OpenAI API spec
type ClientTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
}

func (s *ClientTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
}

func (s *ClientTestSuite) TearDownTest() {
	s.cancel()
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) mock(fn http.HandlerFunc, opts ...Option) (Client, *httptest.Server) {
	srv := httptest.NewServer(fn)
	opts = append([]Option{WithBaseURL(srv.URL)}, opts...)
	cli := New(opts...)
	return cli, srv
}

func (s *ClientTestSuite) TestErrorEncoding() {
	want := Error{
		Type:    "invalid_request_error",
		Message: "message of error",
	}
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{
			"error": {
			  "message": "message of error",
			  "type": "invalid_request_error"
			}
		}`))
	})
	defer srv.Close()
	result, err := client.GenerateChatCompletion(s.ctx, &ChatCompletionParams{})
	s.Nil(result)
	s.Error(err)
	s.Equal(want, err)
}

func (s *ClientTestSuite) TestAuth() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("Bearer api_key", r.Header.Get("Authorization"))
		s.Equal("org_id", r.Header.Get("OpenAI-Organization"))
	}, WithAPIKey("api_key"), WithOrgID("org_id"))
	defer srv.Close()
	client.GenerateChatCompletion(s.ctx, &ChatCompletionParams{})
}

func (s *ClientTestSuite) TestContentType() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("application/json", r.Header.Get("Content-Type"))
	})
	defer srv.Close()
	client.GenerateChatCompletion(s.ctx, &ChatCompletionParams{})
}

func (s *ClientTestSuite) TestGenerateChatCompletion() {
	want := ChatCompletionResult{
		Choices: []ChatCompletionChoice{{
			FinishedReason: "stop",
			Index:          0,
			Message: ChatCompletionMessage{
				Role:    "assistant",
				Content: "Hello",
			},
		}},
	}
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/v1/chat/completions", r.URL.Path)
		s.Equal(http.MethodPost, r.Method)
		s.Equal("Bearer api_key", r.Header.Get("Authorization"))
		s.Equal("org_id", r.Header.Get("OpenAI-Organization"))
		s.Equal("application/json", r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		s.NoError(err)
		s.JSONEq(`{
			"model": "gpt-3.5-turbo",
			"messages": [
				{
					"role": "user",
					"content": "Hello"
				},
				{
					"role": "system",
					"content": "Some system message."
				}
			]
		}`, string(body))
		w.Write([]byte(`{
			"choices": [
				{
					"finish_reason": "stop",
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hello"
					}
				}
			]
		}`))
	}, WithAPIKey("api_key"), WithOrgID("org_id"))
	defer srv.Close()
	result, err := client.GenerateChatCompletion(s.ctx, &ChatCompletionParams{
		Model: "gpt-3.5-turbo",
		Messages: []ChatCompletionMessage{{
			Role:    "user",
			Content: "Hello",
		}, {
			Role:    "system",
			Content: "Some system message.",
		}},
	})
	s.NoError(err)
	s.Equal(&want, result)
}
