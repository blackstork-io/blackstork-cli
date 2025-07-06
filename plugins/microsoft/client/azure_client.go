package client

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/blackstork-io/fabric/internal/plugin/plugindata"
)

const (
	baseURLAzure         = "https://management.azure.com"
	apiVersionAzure      = "2023-11-01"
	defaultPageSizeAzure = 200
)

type azureClient struct {
	accessToken string
	baseURL     string
	client      *http.Client
}

func NewAzureClient(accessToken string) *azureClient {
	return &azureClient{
		accessToken: accessToken,
		client:      &http.Client{},
		baseURL:     baseURLAzure,
	}
}

func (client *azureClient) prepare(r *http.Request) {
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.accessToken))
	q := r.URL.Query()
	q.Add("api-version", apiVersionAzure)
	r.URL.RawQuery = q.Encode()
}

func (client *azureClient) fetchURL(ctx context.Context, requestUrl *url.URL) (result plugindata.Data, err error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, requestUrl.String(), nil)
	if err != nil {
		return
	}
	client.prepare(r)
	slog.DebugContext(ctx, "Fetching an URL from API", "url", requestUrl.String())
	res, err := client.client.Do(r)
	if err != nil {
		return
	}
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read the results: %s", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "Error received from Azure API", "status_code", res.StatusCode, "body", string(raw))
		err = fmt.Errorf("Microsoft Azure API returned status code: %d", res.StatusCode)
		return
	}
	result, err = plugindata.UnmarshalJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %s", err)
	}
	return
}

func (client *azureClient) QueryObjects(
	ctx context.Context,
	endpoint string,
	queryParams url.Values,
	size int,
) (result plugindata.List, err error) {
	urlStr := client.baseURL + endpoint
	requestUrl, err := url.Parse(urlStr)
	if err != nil {
		return
	}

	if queryParams == nil {
		queryParams = url.Values{}
	}

	limit := min(size, defaultPageSizeAzure)
	queryParams.Set("$top", strconv.Itoa(limit))

	requestUrl.RawQuery = queryParams.Encode()

	var totalCount int = -1
	var response plugindata.Data

	objects := make(plugindata.List, 0)

	for {
		slog.DebugContext(ctx, "Fetching a page from Azure API", "url", requestUrl.String())
		response, err = client.fetchURL(ctx, requestUrl)
		if err != nil {
			slog.ErrorContext(ctx, "Error while fetching objects", "url", requestUrl.String(), "error", err)
			return nil, err
		}

		resultMap, ok := response.(plugindata.Map)
		if !ok {
			return nil, fmt.Errorf("unexpected result type: %T", response)
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
			ctx, "Objects fetched from Azure API",
			"fetched_overall", len(objects),
			"fetched", len(objectsPage),
			"total_available", totalCount,
			"to_fetch_overall", size,
		)

		objects = append(objects, objectsPage...)
		if len(objects) >= size {
			break
		}

		nextLink, ok := resultMap["nextLink"]
		if !ok && nextLink == nil {
			break
		}
		requestUrlRaw, ok := nextLink.(plugindata.String)
		if !ok {
			return nil, fmt.Errorf("unexpected value type for `nextLink`: %T", requestUrlRaw)
		}
		requestUrl, err = url.Parse(string(requestUrlRaw))
		if err != nil {
			slog.DebugContext(ctx, "Can't parse the next link in Microsoft Graph API response", "value", requestUrlRaw)
			return nil, err
		}
	}

	objectsToReturn := objects[:min(len(objects), size)]
	return objectsToReturn, nil
}
