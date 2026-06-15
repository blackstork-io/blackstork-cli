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
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
)

const (
	baseURLGraph         = "https://graph.microsoft.com"
	defaultPageSizeGraph = 50
)

type graphClient struct {
	accessToken string
	apiVersion  string
	client      *http.Client
}

func NewGraphClient(accessToken, apiVersion string) *graphClient {
	return &graphClient{
		accessToken: accessToken,
		apiVersion:  apiVersion,
		client:      &http.Client{},
	}
}

func (client *graphClient) prepare(r *http.Request) {
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.accessToken))
	r.Header.Set("ConsistencyLevel", "eventual")
}

func (client *graphClient) fetchURL(ctx context.Context, requestUrl *url.URL) (result plugindata.Data, err error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, requestUrl.String(), nil)
	if err != nil {
		return result, err
	}
	client.prepare(r)
	slog.DebugContext(ctx, "Fetching an URL from API", "url", requestUrl.String())
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
		slog.ErrorContext(ctx, "Error received from Microsoft Graph API", "status_code", res.StatusCode, "body", string(raw))
		err = fmt.Errorf("Microsoft Graph client returned status code: %d", res.StatusCode)
		return result, err
	}

	result, err = plugindata.UnmarshalJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %s", err)
	}
	return result, err
}

func (client *graphClient) QueryObjects(
	ctx context.Context,
	endpoint string,
	queryParams url.Values,
	size int,
) (result plugindata.List, err error) {
	objects := make(plugindata.List, 0)

	urlStr := baseURLGraph + fmt.Sprintf("/%s%s", client.apiVersion, endpoint)
	requestUrl, err := url.Parse(urlStr)
	if err != nil {
		return result, err
	}

	if queryParams == nil {
		queryParams = url.Values{}
	}

	// limit := min(size, defaultPageSizeGraph)
	// $top doesn't work for managedDevices
	// queryParams.Set("$top", strconv.Itoa(limit))
	// queryParams.Set("$count", "true")
	requestUrl.RawQuery = queryParams.Encode()

	totalCount := -1
	var response plugindata.Data

	for {

		if totalCount > 0 {
			queryParams.Set("$skip", strconv.Itoa(len(objects)))
			requestUrl.RawQuery = queryParams.Encode()
		}

		slog.DebugContext(ctx, "Fetching a page from Microsoft Graph API", "url", requestUrl.String())
		response, err = client.fetchURL(ctx, requestUrl)
		if err != nil {
			slog.ErrorContext(ctx, "Error while fetching objects", "url", requestUrl.String(), "error", err)
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
			slog.DebugContext(ctx, "No `@odata.nextLink` found in the response")

			if totalCount < 0 {
				slog.DebugContext(ctx, "Total count is not known, breaking")
				break
			}

			// Check totalCount only if there is no nextLink -- sometimes the response has
			// the count set to the $top value and nextLink is present
			if totalCount > 0 && len(objects) >= totalCount {
				break
			}

		} else {
			requestUrlRaw, ok := nextLink.(plugindata.String)
			if !ok {
				return nil, fmt.Errorf("unexpected value type for `@odata.nextLink`: %T", requestUrlRaw)
			}
			requestUrl, err = url.Parse(string(requestUrlRaw))
			if err != nil {
				slog.DebugContext(ctx, "Can't parse the next link in Microsoft Graph API response", "value", requestUrlRaw)
				return nil, err
			}
		}
	}

	objectsToReturn := objects[:min(len(objects), size)]
	return objectsToReturn, nil
}

func (client *graphClient) QueryObject(
	ctx context.Context,
	endpoint string,
	queryParams url.Values,
) (result plugindata.Data, err error) {
	urlStr := baseURLGraph + fmt.Sprintf("/%s%s", client.apiVersion, endpoint)
	requestUrl, err := url.Parse(urlStr)
	if err != nil {
		return result, err
	}

	if queryParams == nil {
		queryParams = url.Values{}
	}
	requestUrl.RawQuery = queryParams.Encode()

	response, err := client.fetchURL(ctx, requestUrl)
	if err != nil {
		slog.ErrorContext(ctx, "Error while fetching an object", "url", requestUrl.String(), "error", err)
		return nil, err
	}

	return response, nil
}
