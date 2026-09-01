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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

// HTTPClientTestSuite is a test suite for contract testing http data source
type HTTPClientTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
}

func (s *HTTPClientTestSuite) SetupTest() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	s.ctx, s.cancel = context.WithCancel(context.Background())
}

func (s *HTTPClientTestSuite) TearDownTest() {
	s.cancel()
}

func TestHTTPClientTestSuite(t *testing.T) {
	suite.Run(t, new(HTTPClientTestSuite))
}

func (s *HTTPClientTestSuite) mock(fn http.HandlerFunc) *httptest.Server {
	srv := httptest.NewServer(fn)
	return srv
}

func (s *HTTPClientTestSuite) TestErrors() {
	srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	defer srv.Close()

	req := Request{
		URL:        srv.URL,
		Method:     "GET",
		Timeout:    time.Duration(1) * time.Second,
		SkipVerify: true,
		Headers:    make(map[string]string),
	}

	result, err := SendRequest(s.ctx, &req)
	s.Nil(result)
	s.Error(err)

	expectedError := "the server responded with status code 429"
	s.Contains(err.Error(), expectedError)
}

func (s *HTTPClientTestSuite) TestBasicAuth() {
	username := "test-user"
	password := "test-password"
	expectedToken := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))

	srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(
			fmt.Sprintf("Basic %s", expectedToken),
			r.Header.Get("Authorization"),
		)
	})
	defer srv.Close()

	req := Request{
		URL:               srv.URL,
		Method:            "GET",
		Timeout:           time.Duration(1) * time.Second,
		SkipVerify:        true,
		Headers:           make(map[string]string),
		BasicAuthUsername: StringPtr(username),
		BasicAuthPassword: StringPtr(password),
	}

	_, err := SendRequest(s.ctx, &req)
	s.NoError(err)
}

func (s *HTTPClientTestSuite) TestHeadersBody() {
	method := "POST"
	headers := map[string]string{
		"X-Custom":      "foo",
		"Authorization": "bar",
	}
	body := "test-body"

	srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(method, r.Method)
		s.Equal("foo", r.Header.Get("X-Custom"))
		s.Equal("bar", r.Header.Get("Authorization"))

		actualBodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			s.T().Fatalf("Can't read the body of the request")
		}
		s.Equal(body, string(actualBodyBytes))
	})
	defer srv.Close()

	req := Request{
		URL:               srv.URL,
		Method:            method,
		Timeout:           time.Duration(1) * time.Second,
		SkipVerify:        true,
		Headers:           headers,
		Body:              StringPtr(body),
		BasicAuthUsername: StringPtr("a"),
		BasicAuthPassword: StringPtr("b"),
	}

	_, err := SendRequest(s.ctx, &req)
	s.NoError(err)
}

func (s *HTTPClientTestSuite) TestFetchHTTPDataJSON() {
	method := "GET"
	version := "9.9.9"
	insecure := false
	expectedData := []byte(`{
		"foo": 1,
		"bar": [1, "a", {"b": 2}]
	}`)
	username := "test-user"
	password := "test-password"
	expectedToken := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))

	srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(method, r.Method)
		s.Equal(
			fmt.Sprintf("Basic %s", expectedToken),
			r.Header.Get("Authorization"),
		)
		s.Equal(
			fmt.Sprintf("fabric-data-http/%s", version),
			r.Header.Get("User-Agent"),
		)
		header := w.Header()
		header["Content-Type"] = []string{"application/json; charset=utf-8"}
		w.Write(expectedData)
	})
	defer srv.Close()

	ctx := context.Background()

	params := plugin.RetrieveDataParams{
		Args: dataspec.NewBlock(
			[]string{"args"},
			map[string]cty.Value{
				"url":      cty.StringVal(srv.URL),
				"method":   cty.StringVal(method),
				"insecure": cty.BoolVal(insecure),
				"timeout":  cty.StringVal("1s"),
				"headers":  cty.NullVal(cty.Map(cty.String)),
				"body":     cty.NullVal(cty.String),
			},
			dataspec.NewBlock(
				[]string{"basic_auth"},
				map[string]cty.Value{
					"username": cty.StringVal(username),
					"password": cty.StringVal(password),
				},
			),
		),
	}

	data, diags := fetchHTTPData(ctx, &params, version)
	s.Nil(diags)

	actualJson, err := json.Marshal(data)
	s.Nil(err)
	s.JSONEq(string(expectedData), string(actualJson))
}

func (s *HTTPClientTestSuite) TestFetchHTTPDataCSV() {
	method := "GET"
	version := "9.9.9"
	insecure := false

	csvData := []byte(strings.Join([]string{
		"foo,bar",
		"1,xyz",
	}, "\n"))

	username := "test-user"
	password := "test-password"
	expectedToken := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))

	srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(method, r.Method)
		s.Equal(
			fmt.Sprintf("Basic %s", expectedToken),
			r.Header.Get("Authorization"),
		)
		s.Equal(
			fmt.Sprintf("fabric-data-http/%s", version),
			r.Header.Get("User-Agent"),
		)
		header := w.Header()
		header["Content-Type"] = []string{"text/csv; charset=utf-8"}
		w.Write(csvData)
	})
	defer srv.Close()

	ctx := context.Background()

	params := plugin.RetrieveDataParams{
		Args: dataspec.NewBlock(
			[]string{"args"},
			map[string]cty.Value{
				"url":      cty.StringVal(srv.URL),
				"method":   cty.StringVal(method),
				"insecure": cty.BoolVal(insecure),
				"timeout":  cty.StringVal("1s"),
				"headers":  cty.NullVal(cty.Map(cty.String)),
				"body":     cty.NullVal(cty.String),
			},
			dataspec.NewBlock(
				[]string{"basic_auth"},
				map[string]cty.Value{
					"username": cty.StringVal(username),
					"password": cty.StringVal(password),
				},
			),
		),
	}

	data, diags := fetchHTTPData(ctx, &params, version)

	s.Nil(diags, "Error while fetching data: %s", diags)

	s.Equal(
		plugindata.List{
			plugindata.Map{
				"foo": plugindata.Number(1),
				"bar": plugindata.String("xyz"),
			},
		},
		data,
	)
}

func (s *HTTPClientTestSuite) TestFetchHTTPDataUnknownType() {
	method := "GET"
	version := "9.9.9"
	insecure := false

	unknownData := []byte("foobar")

	username := "test-user"
	password := "test-password"
	expectedToken := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))

	srv := s.mock(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(method, r.Method)
		s.Equal(
			fmt.Sprintf("Basic %s", expectedToken),
			r.Header.Get("Authorization"),
		)
		s.Equal(
			fmt.Sprintf("fabric-data-http/%s", version),
			r.Header.Get("User-Agent"),
		)
		header := w.Header()
		header["Content-Type"] = []string{"application/foobar"}
		w.Write(unknownData)
	})
	defer srv.Close()

	ctx := context.Background()

	params := plugin.RetrieveDataParams{
		Args: dataspec.NewBlock(
			[]string{"args"},
			map[string]cty.Value{
				"url":      cty.StringVal(srv.URL),
				"method":   cty.StringVal(method),
				"insecure": cty.BoolVal(insecure),
				"timeout":  cty.StringVal("1s"),
				"headers":  cty.NullVal(cty.Map(cty.String)),
				"body":     cty.NullVal(cty.String),
			},
			dataspec.NewBlock(
				[]string{"basic_auth"},
				map[string]cty.Value{
					"username": cty.StringVal(username),
					"password": cty.StringVal(password),
				},
			),
		),
	}

	data, diags := fetchHTTPData(ctx, &params, version)

	s.Nil(diags, "Error while fetching data: %s", diags)
	s.Equal(plugindata.String(unknownData), data)
}
