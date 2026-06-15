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
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugin/plugintest"
)

func Test_makeRSSDataSchema(t *testing.T) {
	t.Parallel()
	t.Run("basic", func(t *testing.T) {
		t.Parallel()

		assert := assert.New(t)
		schema := makeRSSDataSource()
		assert.Nil(schema.Config)
		assert.NotNil(schema.Args)
		assert.NotNil(schema.DataFunc)
	})
}

func makeTestRssServer() (baseAddr string, closer func()) {
	mux := http.NewServeMux()
	mux.HandleFunc("/up", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	mux.Handle("/basic-auth-redir", http.RedirectHandler(baseAddr+"basic-auth", http.StatusMovedPermanently))
	mux.HandleFunc("/basic-auth", func(w http.ResponseWriter, r *http.Request) {
		un, pw, ok := r.BasicAuth()
		if !ok || un != "user" || pw != "pass" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		r.RequestURI = strings.ReplaceAll(r.RequestURI, "/basic-auth", "/data/basic.rss")
		var err error
		r.URL, err = url.ParseRequestURI(r.RequestURI)
		if err != nil {
			panic(err)
		}
		mux.ServeHTTP(w, r)
	})

	mux.HandleFunc("/content/2", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "test-content")
	})

	path, err := filepath.Abs("./testdata/rss/")
	if err != nil {
		panic(err)
	}

	srv := httptest.NewServer(mux)

	mux.HandleFunc("/data/", func(w http.ResponseWriter, r *http.Request) {
		// http.StripPrefix("/data/", http.FileServerFS(os.DirFS(path)))
		dataFile := strings.TrimPrefix(r.URL.Path, "/data/")
		dataFilePath := filepath.Join(path, dataFile)
		data, err := os.ReadFile(dataFilePath)
		if err != nil {
			panic(err)
		}

		renderedData := strings.NewReplacer("{address}", srv.URL).Replace(string(data))
		fmt.Fprintln(w, renderedData)
		slog.Info("Inserting server address in the feed data", "address", srv.URL)
	})

	close := srv.Close
	defer func() {
		close()
	}()
	baseAddr = srv.URL + "/"
	resp, err := http.Get(baseAddr + "up")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if http.StatusOK != resp.StatusCode {
		panic("resp status code is wrong")
	}

	closer = close
	close = func() {}
	return baseAddr, closer
}

func Test_fetchRSSData(t *testing.T) {
	t.Parallel()

	addr, close := makeTestRssServer()
	defer close()

	type result struct {
		Data  plugindata.Data
		Diags diagnostics.Diag
	}

	ValidRssData := plugindata.Map{
		"description":   plugindata.String("This is an example of an RSS feed"),
		"link":          plugindata.String("http://www.example.com/main.html"),
		"pub_date":      plugindata.String("Sun, 6 Sep 2009 16:20:12 +0000"),
		"title":         plugindata.String("RSS Title"),
		"pub_timestamp": plugindata.Number(1252254012),
		"items": plugindata.List{
			plugindata.Map{
				"description":   plugindata.String("Here is some text containing an interesting description."),
				"guid":          plugindata.String("4824db5b-6278-48bd-9657-46a66de3dc1a"),
				"link":          plugindata.String(addr + "content/2"),
				"pub_date":      plugindata.String("Tue, 8 Sep 2009 22:00:00 +0000"),
				"title":         plugindata.String("Example entry 2"),
				"pub_timestamp": plugindata.Number(1252447200),
				"content":       plugindata.String(""),
			},
			plugindata.Map{
				"description":   plugindata.String("Here is some text containing an interesting description."),
				"guid":          plugindata.String("7bd204c6-1655-4c27-aeee-53f933c5395f"),
				"link":          plugindata.String(addr + "content/1"),
				"pub_date":      plugindata.String("Sun, 6 Sep 2009 16:20:23 +0000"),
				"title":         plugindata.String("Example entry 1"),
				"pub_timestamp": plugindata.Number(1252254023),
				"content":       plugindata.String(""),
			},
		},
	}

	ValidRssDataAfterBeforeTimeFiltered := plugindata.Map{
		"description":   plugindata.String("This is an example of an RSS feed"),
		"link":          plugindata.String("http://www.example.com/main.html"),
		"pub_date":      plugindata.String("Sun, 6 Sep 2009 16:20:12 +0000"),
		"title":         plugindata.String("RSS Title"),
		"pub_timestamp": plugindata.Number(1252254012),
		"items":         plugindata.List{},
	}

	ValidRssDataAfterTimeFiltered := plugindata.Map{
		"description":   plugindata.String("This is an example of an RSS feed"),
		"link":          plugindata.String("http://www.example.com/main.html"),
		"pub_date":      plugindata.String("Sun, 6 Sep 2009 16:20:12 +0000"),
		"title":         plugindata.String("RSS Title"),
		"pub_timestamp": plugindata.Number(1252254012),
		"items": plugindata.List{
			plugindata.Map{
				"description":   plugindata.String("Here is some text containing an interesting description."),
				"guid":          plugindata.String("4824db5b-6278-48bd-9657-46a66de3dc1a"),
				"link":          plugindata.String(addr + "content/2"),
				"pub_date":      plugindata.String("Tue, 8 Sep 2009 22:00:00 +0000"),
				"title":         plugindata.String("Example entry 2"),
				"pub_timestamp": plugindata.Number(1252447200),
				"content":       plugindata.String(""),
			},
		},
	}

	ValidRssDataBeforeimeFiltered := plugindata.Map{
		"description":   plugindata.String("This is an example of an RSS feed"),
		"link":          plugindata.String("http://www.example.com/main.html"),
		"pub_date":      plugindata.String("Sun, 6 Sep 2009 16:20:12 +0000"),
		"title":         plugindata.String("RSS Title"),
		"pub_timestamp": plugindata.Number(1252254012),
		"items": plugindata.List{
			plugindata.Map{
				"description":   plugindata.String("Here is some text containing an interesting description."),
				"guid":          plugindata.String("7bd204c6-1655-4c27-aeee-53f933c5395f"),
				"link":          plugindata.String(addr + "content/1"),
				"pub_date":      plugindata.String("Sun, 6 Sep 2009 16:20:23 +0000"),
				"title":         plugindata.String("Example entry 1"),
				"pub_timestamp": plugindata.Number(1252254023),
				"content":       plugindata.String(""),
			},
		},
	}

	ValidRssWithContent := plugindata.Map{
		"description":   plugindata.String("This is an example of an RSS feed"),
		"link":          plugindata.String("http://www.example.com/main.html"),
		"pub_date":      plugindata.String("Sun, 6 Sep 2009 16:20:12 +0000"),
		"title":         plugindata.String("RSS Title"),
		"pub_timestamp": plugindata.Number(1252254012),
		"items": plugindata.List{
			plugindata.Map{
				"description":   plugindata.String("Here is some text containing an interesting description."),
				"guid":          plugindata.String("4824db5b-6278-48bd-9657-46a66de3dc1a"),
				"link":          plugindata.String(addr + "content/2"),
				"pub_date":      plugindata.String("Tue, 8 Sep 2009 22:00:00 +0000"),
				"title":         plugindata.String("Example entry 2"),
				"pub_timestamp": plugindata.Number(1252447200),
				// Note, go-readability wraps content in a `div`. We can use `article.TextContent` if
				// we want only the text content returned
				"content": plugindata.String("<div id=\"readability-page-1\">test-content\n</div>"),
			},
			plugindata.Map{
				"description":   plugindata.String("Here is some text containing an interesting description."),
				"guid":          plugindata.String("7bd204c6-1655-4c27-aeee-53f933c5395f"),
				"link":          plugindata.String(addr + "content/1"),
				"pub_date":      plugindata.String("Sun, 6 Sep 2009 16:20:23 +0000"),
				"title":         plugindata.String("Example entry 1"),
				"pub_timestamp": plugindata.Number(1252254023),
				"content":       plugindata.String(""),
			},
		},
	}

	ValidAtomData := plugindata.Map{
		"description": plugindata.String("A subtitle."),
		"link":        plugindata.String("http://example.org/"),
		"pub_date":    plugindata.String(""),
		"title":       plugindata.String("Example Feed"),
		"items": plugindata.List{
			plugindata.Map{
				"description":   plugindata.String("Some text."),
				"guid":          plugindata.String("urn:uuid:1225c695-cfb8-4ebb-aaaa-80da344efa6a"),
				"link":          plugindata.String("http://example.org/2003/12/13/atom03"),
				"pub_date":      plugindata.String("2003-11-09T17:23:02Z"),
				"pub_timestamp": plugindata.Number(1068398582),
				"title":         plugindata.String("Atom-Powered Robots Run Amok"),
				"content":       plugindata.String("\n\t\t\t\t<p>This is the entry content.</p>\n\t\t\t"),
			},
		},
	}

	tt := []struct {
		name              string
		url               string
		items_after       string
		items_before      string
		fill_in_content   bool
		max_items_to_fill int
		auth              *gofeed.Auth
		expected          result
	}{
		{
			name: "valid_rss",
			url:  "data/basic.rss",
			expected: result{
				Data: ValidRssData,
			},
		},
		{
			name: "valid_atom",
			url:  "data/basic.atom",
			expected: result{
				Data: ValidAtomData,
			},
		},
		{
			name:        "valid_rss_items_after",
			url:         "data/basic.rss",
			items_after: "2009-09-07T00:00:00Z",
			expected: result{
				Data: ValidRssDataAfterTimeFiltered,
			},
		},
		{
			name:         "valid_rss_items_after_before",
			url:          "data/basic.rss",
			items_after:  "2009-09-07T00:00:00Z",
			items_before: "2009-09-08T00:00:00Z",
			expected: result{
				Data: ValidRssDataAfterBeforeTimeFiltered,
			},
		},
		{
			name:         "valid_rss_items_before",
			url:          "data/basic.rss",
			items_before: "2009-09-08T00:00:00Z",
			expected: result{
				Data: ValidRssDataBeforeimeFiltered,
			},
		},
		{
			name:              "valid_rss_with_fill_in",
			url:               "data/basic.rss",
			fill_in_content:   true,
			max_items_to_fill: 1,
			expected: result{
				Data: ValidRssWithContent,
			},
		},
		{
			name: "valid_auth",
			url:  "basic-auth",
			auth: &gofeed.Auth{
				Username: "user",
				Password: "pass",
			},

			expected: result{
				Data: ValidRssData,
			},
		},
		{
			name: "invalid_auth",
			url:  "basic-auth",
			auth: &gofeed.Auth{
				Username: "invalid_user",
				Password: "invalid_pass",
			},

			expected: result{
				Diags: diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Failed to fetch the feed `%s/basic-auth`", strings.TrimSuffix(addr, "/")),
					Detail:   "http error: 401 Unauthorized",
				}},
			},
		},
		{
			name: "empty_auth",
			url:  "basic-auth",
			auth: &gofeed.Auth{},
			expected: result{
				Diags: diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Failed to fetch the feed `%s/basic-auth`", strings.TrimSuffix(addr, "/")),
					Detail:   "http error: 401 Unauthorized",
				}},
			},
		},
		{
			name: "incomplete_auth",
			url:  "basic-auth",
			auth: &gofeed.Auth{
				Username: "user",
			},
			expected: result{
				Diags: diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Failed to fetch the feed `%s/basic-auth`", strings.TrimSuffix(addr, "/")),
					Detail:   "http error: 401 Unauthorized",
				}},
			},
		},
		{
			name: "absent_auth",
			url:  "basic-auth",
			expected: result{
				Diags: diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Failed to fetch the feed `%s/basic-auth`", strings.TrimSuffix(addr, "/")),
					Detail:   "http error: 401 Unauthorized",
				}},
			},
		},
		{
			name: "valid_auth_redir",
			url:  "basic-auth-redir",
			auth: &gofeed.Auth{
				Username: "user",
				Password: "pass",
			},
			expected: result{
				Data: ValidRssData,
			},
		},
		{
			name: "invalid_path",
			url:  "does_not_exist.rss",
			expected: result{
				Diags: diagnostics.Diag{{
					Severity: hcl.DiagError,
					Summary: fmt.Sprintf(
						"Failed to fetch the feed `%s/does_not_exist.rss`",
						strings.TrimSuffix(addr, "/"),
					),
					Detail: "http error: 404 Not Found",
				}},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			slog.Info("Running test case", "name", tc.name)

			assert := assert.New(t)

			p := &plugin.Schema{
				DataSources: plugin.DataSources{
					"rss": makeRSSDataSource(),
				},
			}

			dec := plugintest.NewTestDecoder(t, p.DataSources["rss"].Args).
				SetAttr("url", cty.StringVal(addr+tc.url))

			if tc.items_after != "" {
				dec = dec.SetAttr("items_after", cty.StringVal(tc.items_after))
			}
			if tc.items_before != "" {
				dec = dec.SetAttr("items_before", cty.StringVal(tc.items_before))
			}
			dec = dec.SetAttr("fill_in_content", cty.BoolVal(tc.fill_in_content))

			if tc.max_items_to_fill != 0 {
				dec = dec.SetAttr("max_items_to_fill", cty.NumberIntVal(int64(tc.max_items_to_fill)))
			}

			if tc.auth != nil {
				dec.AppendBody(fmt.Sprintf(`
					basic_auth {
						username = "%s"
						password = "%s"
					}
					`, tc.auth.Username, tc.auth.Password))
			}

			params := &plugin.RetrieveDataParams{Args: dec.Decode()}

			data, diags := p.RetrieveData(context.Background(), "rss", params)
			assert.Equal(tc.expected, result{data, diags})
		})
	}
}
