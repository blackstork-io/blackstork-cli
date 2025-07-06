package client

import (
	"strings"
)

func String(s string) *string {
	return &s
}

func Int(i int) *int {
	return &i
}

type Error struct {
	ErrorMessages []string `json:"errorMessages"`
}

func (err *Error) Error() string {
	return strings.Join(err.ErrorMessages, " ")
}

type SearchIssuesReq struct {
	Expand        *string  `json:"expand,omitempty"`
	Fields        []string `json:"fields,omitempty"`
	JQL           *string  `json:"jql,omitempty"`
	Properties    []string `json:"properties,omitempty"`
	NextPageToken *string  `json:"nextPageToken,omitempty"`
	MaxResults    *int     `json:"maxResults,omitempty"`
}

type SearchIssuesRes struct {
	NextPageToken *string `json:"nextPageToken,omitempty"`
	Issues        []any   `json:"issues"`
}
