package client

import "fmt"

type ChatCompletionParams struct {
	Model    string                  `json:"model"`
	Messages []ChatCompletionMessage `json:"messages"`
}

type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResult struct {
	Choices []ChatCompletionChoice `json:"choices"`
}

type ChatCompletionChoice struct {
	FinishedReason string                `json:"finish_reason"`
	Index          int                   `json:"index"`
	Message        ChatCompletionMessage `json:"message"`
}

type Error struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("openai[%s]: %s", e.Type, e.Message)
}

type ErrorResponse struct {
	Error Error `json:"error"`
}
