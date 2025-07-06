package hubapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"time"

	"google.golang.org/protobuf/proto"

	pluginapiv1 "github.com/blackstork-io/fabric/internal/plugin/pluginapi/v1"
)

const (
	contentJSON = "application/json"
	contentPB   = "application/protobuf"

	headerAuthorization = "Authorization"
	headerUserAgent     = "User-Agent"
	headerContentType   = "Content-Type"
	headerAccept        = "Accept"

	userAgent = "blackstork"
)

type Client interface {
	CreateDocument(ctx context.Context, params *DocumentParams) (*Document, error)
	UploadDocumentContent(ctx context.Context, documentID string, content *pluginapiv1.Content) (*DocumentContent, error)
}

type client struct {
	apiURL, apiToken string
	version          string
	httpClient       *http.Client
}

func NewClient(apiURL, apiToken, version string) Client {
	return &client{
		apiURL:   apiURL,
		apiToken: apiToken,
		version:  version,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (cli *client) CreateDocument(ctx context.Context, params *DocumentParams) (*Document, error) {
	url, err := cli.makeURL("/api/v1/documents")
	if err != nil {
		return nil, err
	}

	data, err := cli.callJSON(ctx, http.MethodPost, url, params)
	if err != nil {
		return nil, err
	}

	var result Document

	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response data: %w", err)
	}

	return &result, nil
}

func (cli *client) UploadDocumentContent(ctx context.Context, documentID string, content *pluginapiv1.Content) (*DocumentContent, error) {
	url, err := cli.makeURL("/api/v1/documents/", documentID, "/content")
	if err != nil {
		return nil, err
	}

	data, err := cli.uploadPB(ctx, url, content)
	if err != nil {
		return nil, err
	}

	var result DocumentContent
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response data: %w", err)
	}

	return &result, nil
}

func (cli *client) addCommonHeaders(r *http.Request) {
	r.Header.Add(headerAuthorization, fmt.Sprintf("Bearer %s", cli.apiToken))
	r.Header.Add(headerUserAgent, fmt.Sprintf("%s/%s", userAgent, cli.version))
}

func (cli *client) makeURL(path ...string) (string, error) {
	return url.JoinPath(cli.apiURL, path...)
}

func (cli *client) callJSON(ctx context.Context, method, url string, params any) (json.RawMessage, error) {
	body, err := json.Marshal(request{params})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	cli.addCommonHeaders(req)

	req.Header.Add(headerContentType, contentJSON)
	req.Header.Add(headerAccept, contentJSON)

	res, err := cli.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	mediaType, _, err := mime.ParseMediaType(res.Header.Get(headerContentType))
	if err != nil {
		return nil, fmt.Errorf("failed to parse response content type: %w", err)
	}

	if mediaType != contentJSON {
		return nil, fmt.Errorf("invalid response content type: %s", mediaType)
	}

	var jsonRes response

	err = json.NewDecoder(res.Body).Decode(&jsonRes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if jsonRes.Error != nil {
		return nil, jsonRes.Error
	} else if res.StatusCode >= 400 {
		return nil, fmt.Errorf("invalid status code: %d", res.StatusCode)
	}

	return jsonRes.Data, nil
}

func (cli *client) uploadPB(ctx context.Context, url string, data proto.Message) (json.RawMessage, error) {
	body, err := proto.Marshal(data)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	cli.addCommonHeaders(req)

	contentType := mime.FormatMediaType(contentPB, map[string]string{
		"proto": string(proto.MessageName(data)),
	})

	req.Header.Add(headerContentType, contentType)
	req.Header.Add(headerAccept, contentJSON)

	res, err := cli.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	resContentType, _, err := mime.ParseMediaType(res.Header.Get(headerContentType))
	if err != nil {
		return nil, fmt.Errorf("failed to parse response content type: %w", err)
	}

	if resContentType != contentJSON {
		return nil, fmt.Errorf("invalid response content type: %s", resContentType)
	}

	var jsonRes response

	err = json.NewDecoder(res.Body).Decode(&jsonRes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if jsonRes.Error != nil {
		return nil, jsonRes.Error
	} else if res.StatusCode >= 400 {
		return nil, fmt.Errorf("invalid status code: %d", res.StatusCode)
	}

	return jsonRes.Data, nil
}
