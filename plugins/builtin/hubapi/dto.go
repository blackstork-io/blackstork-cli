package hubapi

import (
	"encoding/json"
	"strings"
	"time"
)

type request struct {
	Params any `json:"params"`
}

type response struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error *Error          `json:"error,omitempty"`
}

type Error struct {
	Details []*ErrorDetail `json:"details"`
}

type ErrorDetail struct {
	Message string `json:"message"`
}

func (err *Error) Error() string {
	messages := make([]string, len(err.Details))
	for i, detail := range err.Details {
		messages[i] = detail.Message
	}
	return strings.Join(messages, "; ")
}

type Document struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	ContentID *string   `json:"content_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DocumentParams struct {
	Title string `json:"title"`
}

type DocumentContent struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}
