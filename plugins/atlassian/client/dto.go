// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

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
