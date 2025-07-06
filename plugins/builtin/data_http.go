package builtin

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/fabric/internal/builtin/utils"
	"github.com/blackstork-io/fabric/internal/utils/diagnostics"
	u "github.com/blackstork-io/fabric/internal/utils"
	"github.com/blackstork-io/fabric/internal/plugin"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
	"github.com/blackstork-io/fabric/internal/plugin/dataspec/constraint"
	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

func makeHTTPDataSource(version string) *plugin.DataSource {
	return &plugin.DataSource{
		DataFunc: fetchHTTPDataWrapper(version),
		Args: &dataspec.RootSpec{
			Blocks: []*dataspec.BlockSpec{
				{
					Header: dataspec.HeadersSpec{
						dataspec.ExactMatcher{"basic_auth"},
					},
					Doc: u.Dedent(`
						Basic authentication credentials to be used for HTTP request.
					`),
					Attrs: []*dataspec.AttrSpec{
						{
							Name:        "username",
							Type:        cty.String,
							ExampleVal:  cty.StringVal("user@example.com"),
							Constraints: constraint.RequiredNonNull,
						},
						{
							Name:       "password",
							Type:       cty.String,
							ExampleVal: cty.StringVal("passwd"),
							Doc: u.Dedent(`
								Note: avoid storing credentials in the templates. Use environment variables instead.
							`),
							Constraints: constraint.RequiredNonNull,
						},
					},
				},
			},
			Attrs: []*dataspec.AttrSpec{
				{
					Name:        "url",
					Type:        cty.String,
					ExampleVal:  cty.StringVal("https://example.localhost/file.json"),
					Constraints: constraint.RequiredMeaningful,
					Doc:         "URL to fetch data from. Supported schemas are `http` and `https`",
				},
				{
					Name:       "method",
					Type:       cty.String,
					DefaultVal: cty.StringVal("GET"),
					OneOf: []cty.Value{
						cty.StringVal("GET"),
						cty.StringVal("POST"),
						cty.StringVal("HEAD"),
					},
					Doc: "HTTP method for the request. Allowed methods are `GET`, `POST` and `HEAD`",
				},
				{
					Name:       "insecure",
					Type:       cty.Bool,
					DefaultVal: cty.BoolVal(false),
					Doc:        "If set to `true`, disabled verification of the server's certificate.",
				},
				{
					Name:       "timeout",
					Type:       cty.String,
					DefaultVal: cty.StringVal("30s"),
					Doc:        "The duration of a timeout for a request. Accepts numbers, with optional fractions and a unit suffix. For example, valid values would be: 1.5s, 30s, 2m, 2m30s, or 1h",
				},
				{
					Name:       "headers",
					Type:       cty.Map(cty.String),
					DefaultVal: cty.NullVal(cty.Map(cty.String)),
					Doc:        `The headers to be set in a request`,
				},
				{
					Name:       "body",
					Type:       cty.String,
					DefaultVal: cty.NullVal(cty.String),
					Doc:        `Request body`,
				},
			},
		},
		Doc: u.Dedent(`
			Loads data from a URL.

			At the moment, the data source accepts only responses with UTF-8 charset and parses only responses
			with MIME types ` + "`text/csv`" + ` or ` + "`application/json`" + `.

			If MIME type of the response is ` + "`text/csv`" + ` or ` + "`application/json`" + `, the response
			content will be parsed and returned as a JSON structure (similar to the behaviour of CSV and JSON data
			sources). Otherwise, the response content will be returned as text
		`),
	}
}

func StringPtr(s string) *string {
	return &s
}

type Request struct {
	Url               string
	Method            string
	Timeout           time.Duration
	SkipVerify        bool
	Headers           map[string]string
	Body              *string
	BasicAuthUsername *string
	BasicAuthPassword *string
}

type Response struct {
	Body     []byte
	MimeType string
}

func SendRequest(ctx context.Context, log *slog.Logger, r *Request) (*Response, error) {
	var u *url.URL
	var err error

	u, err = url.Parse(r.Url)
	if err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if r.Body != nil {
		reqBody = strings.NewReader(*r.Body)
	}

	request, err := http.NewRequestWithContext(ctx, r.Method, u.String(), reqBody)
	if err != nil {
		return nil, err
	}

	if r.BasicAuthUsername != nil && r.BasicAuthPassword != nil {
		request.SetBasicAuth(*r.BasicAuthUsername, *r.BasicAuthPassword)
	}

	if r.Headers != nil {
		for k, v := range r.Headers {
			request.Header.Set(k, v)
		}
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: r.SkipVerify, //nolint:gosec,G402
	}
	client := &http.Client{Transport: transport, Timeout: r.Timeout}

	log.DebugContext(
		ctx,
		"Sending HTTP request",
		"url", r.Url,
		"method", r.Method,
		"insecure", r.SkipVerify,
		"timeout", r.Timeout,
	)

	res, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the server responded with status code %d", res.StatusCode)
	}

	contentType := res.Header.Get("Content-Type")
	var mimeType string
	if contentType == "" {
		mimeType = "text/plain" // assume `text/plain` if no content type set
	} else {
		mimeType, _, err = mime.ParseMediaType(contentType)
		if err != nil {
			return nil, err
		}
	}

	defer res.Body.Close()

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error while reading response body: %s", err)
	}

	if !utf8.Valid(bytes) {
		return nil, fmt.Errorf("response body is not recognized as UTF-8: %s", err)
	}
	return &Response{Body: bytes, MimeType: mimeType}, nil
}

func fetchHTTPDataWrapper(version string) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (plugindata.Data, diagnostics.Diag) {
		return fetchHTTPData(ctx, params, version)
	}
}

func fetchHTTPData(ctx context.Context, params *plugin.RetrieveDataParams, version string) (plugindata.Data, diagnostics.Diag) {

	log := slog.Default()
	log = log.With("data_source", "http")

	url := params.Args.GetAttrVal("url").AsString()
	method := params.Args.GetAttrVal("method").AsString()
	insecure := params.Args.GetAttrVal("insecure").True()

	log = log.With("url", url)

	timeout, err := time.ParseDuration(params.Args.GetAttrVal("timeout").AsString())
	if err != nil {
		return nil, diagnostics.Diag{
			{
				Severity: hcl.DiagError,
				Summary:  "Failed to parse a timeout duraction value",
				Detail:   err.Error(),
			},
		}
	}

	req := Request{
		Url:        url,
		Method:     method,
		Timeout:    timeout,
		SkipVerify: insecure,
		Headers:    make(map[string]string),
		Body:       nil,
	}

	basicAuth := params.Args.Blocks.GetFirstMatching("basic_auth")
	if basicAuth != nil {
		req.BasicAuthUsername = StringPtr(basicAuth.GetAttrVal("username").AsString())
		req.BasicAuthPassword = StringPtr(basicAuth.GetAttrVal("password").AsString())
	}

	headers := params.Args.GetAttrVal("headers")
	if !headers.IsNull() {
		for k, v := range headers.AsValueMap() {
			req.Headers[k] = v.AsString()
		}
	}

	body := params.Args.GetAttrVal("body")
	if !body.IsNull() && body.AsString() != "" {
		req.Body = StringPtr(body.AsString())
	}

	req.Headers["User-Agent"] = fmt.Sprintf("fabric-data-http/%s", version)

	response, err := SendRequest(ctx, log, &req)
	if err != nil {
		return nil, diagnostics.Diag{
			{
				Severity: hcl.DiagError,
				Summary:  "Failed to fetch data with HTTP request",
				Detail:   err.Error(),
			},
		}
	}
	log.DebugContext(ctx, "Response received", "mime_type", response.MimeType, "body_bytes_count", len(response.Body))

	var result plugindata.Data

	if response.MimeType == "text/csv" {
		reader := csv.NewReader(bytes.NewBuffer(response.Body))
		reader.Comma = ',' // Use `,` as a CSV delimiter by default

		log.DebugContext(ctx, "Parsing fetched data as CSV", "mime-type", response.MimeType)
		result, err = utils.ParseCSVContent(ctx, reader)
		if err != nil {
			return nil, diagnostics.Diag{
				{
					Severity: hcl.DiagError,
					Summary:  "Failed to parse CSV content",
					Detail:   err.Error(),
				},
			}
		}
	} else if response.MimeType == "application/json" {

		log.DebugContext(ctx, "Parsing fetched data as JSON", "mime-type", response.MimeType)

		result, err = plugindata.UnmarshalJSON(response.Body)
		if err != nil {
			return nil, diagnostics.Diag{
				{
					Severity: hcl.DiagError,
					Summary:  "Failed to parse JSON content",
					Detail:   err.Error(),
				},
			}
		}
	} else {
		log.DebugContext(ctx, "Returning fetched data as text", "mime-type", response.MimeType)
		result = plugindata.String(response.Body)
	}
	return result, nil
}
