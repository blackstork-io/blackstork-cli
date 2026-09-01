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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
)

const (
	baseURLSecurity         = "https://api.securitycenter.microsoft.com/api"
	defaultPageSizeSecurity = 200
)

type securityClient struct {
	accessToken string
	client      *http.Client
}

func NewSecurityClient(accessToken string) *securityClient {
	return &securityClient{
		accessToken: accessToken,
		client:      &http.Client{},
	}
}

func (client *securityClient) prepare(r *http.Request) {
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.accessToken))
}

func (client *securityClient) getURL(ctx context.Context, requestURL *url.URL) (result plugindata.Data, err error) {
	slog.DebugContext(ctx, "Sending GET request to an API endpoint", "url", requestURL.String())
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return result, err
	}
	client.prepare(r)
	res, err := client.client.Do(r)
	if err != nil {
		return result, err
	}
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read the results: %s", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		slog.ErrorContext(
			ctx,
			"Error received from Microsoft Graph API",
			"status_code",
			res.StatusCode,
			"body",
			string(raw),
		)
		err = fmt.Errorf("microsoft Graph client returned status code: %d", res.StatusCode)
		return result, err
	}
	result, err = plugindata.UnmarshalJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %s", err)
	}
	return result, err
}

func (client *securityClient) postURL(
	ctx context.Context,
	requestURL *url.URL,
	data plugindata.Data,
) (result plugindata.Data, err error) {
	buff := new(bytes.Buffer)
	err = json.NewEncoder(buff).Encode(data)
	if err != nil {
		return result, err
	}
	slog.DebugContext(ctx, "Sending POST request to an API endpoint", "url", requestURL.String())

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL.String(), buff)
	r.Header.Set("Content-Type", "application/json")
	if err != nil {
		return result, err
	}
	client.prepare(r)
	res, err := client.client.Do(r)
	if err != nil {
		return result, err
	}
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read the results: %s", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "Error received from API", "status_code", res.StatusCode, "body", string(raw))
		if res.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		err = fmt.Errorf("API returned status code %d", res.StatusCode)
		return result, err
	}

	result, err = plugindata.UnmarshalJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %s", err)
	}
	return result, nil
}

func (client *securityClient) QueryObjects(
	ctx context.Context,
	endpoint string,
	queryParams url.Values,
	size int,
) (result plugindata.List, err error) {
	objects := make(plugindata.List, 0)

	urlStr := baseURLSecurity + endpoint
	requestURL, err := url.Parse(urlStr)
	if err != nil {
		return result, err
	}

	if queryParams == nil {
		queryParams = url.Values{}
	}

	limit := min(size, defaultPageSizeSecurity)
	queryParams.Set("$top", strconv.Itoa(limit))

	requestURL.RawQuery = queryParams.Encode()

	totalCount := -1
	var response plugindata.Data

	for {
		slog.DebugContext(ctx, "Fetching a page from Microsoft Graph API", "url", requestURL.String())
		response, err = client.getURL(ctx, requestURL)
		if err != nil {
			slog.ErrorContext(ctx, "Error while fetching objects", "url", requestURL.String(), "error", err)
			return nil, err
		}

		resultMap, ok := response.(plugindata.Map)
		if !ok {
			return nil, fmt.Errorf("unexpected result type: %T", response)
		}

		countRaw, ok := resultMap["@odata.count"]
		if ok {
			totalCount = int(countRaw.(plugindata.Number))
		}

		objectsPageRaw, ok := resultMap["value"]
		if !ok {
			break
		}

		objectsPage, ok := objectsPageRaw.(plugindata.List)
		if !ok {
			return nil, fmt.Errorf("unexpected value type: %T", objectsPageRaw)
		}

		if len(objectsPage) == 0 {
			break
		}

		slog.DebugContext(
			ctx, "Objects fetched from Microsoft Graph API",
			"fetched_overall", len(objects),
			"fetched", len(objectsPage),
			"total_available", totalCount,
			"to_fetch_overall", size,
		)

		objects = append(objects, objectsPage...)
		if len(objects) >= size {
			break
		}

		nextLink, ok := resultMap["@odata.nextLink"]
		if !ok && nextLink == nil {
			break
		}
		requestURLRaw, ok := nextLink.(plugindata.String)
		if !ok {
			return nil, fmt.Errorf("unexpected value type for `@odata.nextLink`: %T", requestURLRaw)
		}
		requestURL, err = url.Parse(string(requestURLRaw))
		if err != nil {
			slog.DebugContext(ctx, "Can't parse the next link in Microsoft Graph API response", "value", requestURLRaw)
			return nil, err
		}
	}

	objectsToReturn := objects[:min(len(objects), size)]
	return objectsToReturn, nil
}

func (client *securityClient) QueryObject(
	ctx context.Context,
	endpoint string,
	queryParams url.Values,
) (result plugindata.Data, err error) {
	urlStr := baseURLSecurity + endpoint
	requestURL, err := url.Parse(urlStr)
	if err != nil {
		return result, err
	}

	if queryParams == nil {
		queryParams = url.Values{}
	}
	requestURL.RawQuery = queryParams.Encode()

	response, err := client.getURL(ctx, requestURL)
	if err != nil {
		slog.ErrorContext(ctx, "Error while fetching an object", "url", requestURL.String(), "error", err)
		return nil, err
	}

	return response, nil
}

func (client *securityClient) RunAdvancedQuery(ctx context.Context, query string) (result plugindata.Data, err error) {
	urlStr := baseURLSecurity + "/advancedqueries/run"
	requestURL, err := url.Parse(urlStr)
	if err != nil {
		return result, err
	}

	body := plugindata.Map{
		"Query": plugindata.String(query),
	}

	response, err := client.postURL(ctx, requestURL, body)
	if err != nil {
		slog.ErrorContext(ctx, "Error while submitting an advanced query", "url", requestURL.String(), "error", err, "query", query)
		return nil, err
	}

	return response, nil
}
