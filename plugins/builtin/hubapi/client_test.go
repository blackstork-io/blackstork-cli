// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package hubapi

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	pluginapiv1 "github.com/blackstork-io/blackstork-cli/plugin/pluginapi/v1"
)

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

func (s *ClientTestSuite) mock(fn http.HandlerFunc, apiToken, version string) (Client, *httptest.Server) {
	srv := httptest.NewServer(fn)
	cli := NewClient(srv.URL, apiToken, version)
	return cli, srv
}

func (s *ClientTestSuite) assertCommonHeaders(r *http.Request, wantToken, wantVersion string) {
	gotToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	s.Equal(wantToken, gotToken, "expected valid token")
	wantUserAgent := fmt.Sprintf("blackstork/%s", wantVersion)
	s.Equal(wantUserAgent, r.Header.Get("User-Agent"))
}

func (s *ClientTestSuite) readAll(r io.Reader) string {
	return string(s.readAllBytes(r))
}

func (s *ClientTestSuite) readAllBytes(r io.Reader) []byte {
	body, err := io.ReadAll(r)
	s.Require().NoError(err)
	return body
}

func (s *ClientTestSuite) makeTime(str string) time.Time {
	ts, err := time.Parse(time.RFC3339, str)
	s.Require().NoError(err)
	return ts
}

func (s *ClientTestSuite) TestCreateDocument() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.assertCommonHeaders(r, "test_token", "v0.0.1-test")
		s.Equal("application/json", r.Header.Get("Content-Type"))
		s.Equal("application/json", r.Header.Get("Accept"))
		s.Equal("/api/v1/documents", r.URL.Path)
		s.Equal(http.MethodPost, r.Method)

		defer r.Body.Close()

		s.JSONEq(`{
			"params": {
				"title": "Test Title"
			}
		}`, s.readAll(r.Body))
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"data": {
				"id": "document_001",
				"title": "Test Title",
				"content_id": null,
				"created_at": "2024-11-09T06:56:25Z",
				"updated_at": "2024-11-09T06:56:25Z"
			}
		}`))
	}, "test_token", "v0.0.1-test")

	defer srv.Close()

	result, err := client.CreateDocument(s.ctx, &DocumentParams{
		Title: "Test Title",
	})
	s.NoError(err)
	s.Equal(&Document{
		ID:        "document_001",
		Title:     "Test Title",
		ContentID: nil,
		CreatedAt: s.makeTime("2024-11-09T06:56:25Z"),
		UpdatedAt: s.makeTime("2024-11-09T06:56:25Z"),
	}, result)
}

func (s *ClientTestSuite) TestCreateDocumentError() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{
			"error": {
				"details": [{
					"message": "Test Error"
				}]
			}
		}`))
	}, "test_token", "v0.0.1-test")

	defer srv.Close()

	result, err := client.CreateDocument(s.ctx, &DocumentParams{
		Title: "Test Title",
	})
	s.Nil(result)

	var apiErr *Error

	s.ErrorAs(err, &apiErr)
	s.Equal(apiErr, &Error{
		Details: []*ErrorDetail{
			{
				Message: "Test Error",
			},
		},
	})
}

func (s *ClientTestSuite) TestUploadDocumentContent() {
	wantContent := &pluginapiv1.Content{
		Value: &pluginapiv1.Content_Section{
			Section: &pluginapiv1.ContentSection{
				Children: []*pluginapiv1.Content{
					{
						Value: &pluginapiv1.Content_Element{
							Element: &pluginapiv1.ContentElement{
								Markdown: []byte("# Hello World"),
							},
						},
					},
				},
			},
		},
	}
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.assertCommonHeaders(r, "test_token", "v0.0.1-test")
		ct, ctParams, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		s.Require().NoError(err)
		s.Equal("application/protobuf", ct)
		s.Equal("pluginapi.v1.Content", ctParams["proto"])
		s.Equal("application/json", r.Header.Get("Accept"))
		s.Equal("/api/v1/documents/document_001/content", r.URL.Path)
		s.Equal(http.MethodPost, r.Method)

		defer r.Body.Close()

		var gotContent pluginapiv1.Content
		s.NoError(proto.Unmarshal(s.readAllBytes(r.Body), &gotContent))
		s.True(proto.Equal(wantContent, &gotContent), "protobuf payload should match")

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"data": {
				"id": "content_001",
				"created_at": "2024-11-09T06:56:25Z"
			}
		}`))
	}, "test_token", "v0.0.1-test")

	defer srv.Close()

	result, err := client.UploadDocumentContent(s.ctx, "document_001", wantContent)
	s.NoError(err)
	s.Equal(&DocumentContent{
		ID:        "content_001",
		CreatedAt: s.makeTime("2024-11-09T06:56:25Z"),
	}, result)
}

func (s *ClientTestSuite) TestUploadDocumentContentError() {
	client, srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{
			"error": {
				"details": [{
					"message": "Test Error"
				}]
			}
		}`))
	}, "test_token", "v0.0.1-test")

	defer srv.Close()

	result, err := client.UploadDocumentContent(s.ctx, "document_001", &pluginapiv1.Content{})
	s.Nil(result)

	var apiErr *Error

	s.ErrorAs(err, &apiErr)
	s.Equal(apiErr, &Error{
		Details: []*ErrorDetail{
			{
				Message: "Test Error",
			},
		},
	})
}
