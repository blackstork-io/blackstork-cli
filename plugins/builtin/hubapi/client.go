// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

// Package hubapi implements the BlackStork Hub API client.
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

	pluginapiv1 "github.com/blackstork-io/blackstork-cli/plugin/pluginapi/v1"
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
	UploadDocument(ctx context.Context, doc *pluginapiv1.Document) (*HubDocument, error)
}

type client struct {
	apiURL, apiToken string
	version          string
	httpClient       *http.Client
}

func NewClient(url, apiToken, version string) Client {
	return &client{
		apiURL:   url,
		apiToken: apiToken,
		version:  version,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type hubResponse struct {
	Data *HubDocument `json:"data"`
}

func (cli *client) UploadDocument(ctx context.Context, doc *pluginapiv1.Document) (*HubDocument, error) {
	url, err := cli.makeURL("/api/documents")
	if err != nil {
		return nil, err
	}

	data, err := cli.uploadPB(ctx, url, doc)
	if err != nil {
		return nil, err
	}

	res := hubResponse{}

	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal hub response: %w", err)
	}

	// data, err := cli.callJSON(ctx, http.MethodPost, url, params)
	// if err != nil {
	// 	return nil, err
	// }
	//
	// var result Document
	//
	// err = json.Unmarshal(data, &result)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to unmarshal response data: %w", err)
	// }

	return res.Data, nil
}

func (cli *client) addCommonHeaders(r *http.Request) {
	r.Header.Add(headerAuthorization, fmt.Sprintf("Bearer %s", cli.apiToken))
	r.Header.Add(headerUserAgent, fmt.Sprintf("%s/%s", userAgent, cli.version))
}

func (cli *client) makeURL(path ...string) (string, error) {
	return url.JoinPath(cli.apiURL, path...)
}

// func (cli *client) callJSON(ctx context.Context, method, url string, params any) (json.RawMessage, error) {
// 	body, err := json.Marshal(request{params})
// 	if err != nil {
// 		return nil, err
// 	}
// 	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	cli.addCommonHeaders(req)
//
// 	req.Header.Add(headerContentType, contentJSON)
// 	req.Header.Add(headerAccept, contentJSON)
//
// 	res, err := cli.httpClient.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	defer res.Body.Close()
//
// 	if res.StatusCode == http.StatusNoContent {
// 		return nil, nil
// 	}
//
// 	mediaType, _, err := mime.ParseMediaType(res.Header.Get(headerContentType))
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to parse response content type: %w", err)
// 	}
//
// 	if mediaType != contentJSON {
// 		return nil, fmt.Errorf("invalid response content type: %s", mediaType)
// 	}
//
// 	var jsonRes response
//
// 	err = json.NewDecoder(res.Body).Decode(&jsonRes)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to parse response: %w", err)
// 	}
//
// 	if jsonRes.Error != nil {
// 		return nil, jsonRes.Error
// 	} else if res.StatusCode >= 400 {
// 		return nil, fmt.Errorf("invalid status code: %d", res.StatusCode)
// 	}
//
// 	return jsonRes.Data, nil
// }

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

	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("error status code received: %d", res.StatusCode)
	}

	resContentType, _, err := mime.ParseMediaType(res.Header.Get(headerContentType))
	if err != nil {
		return nil, fmt.Errorf("failed to parse response content type: %w", err)
	}

	if resContentType != contentJSON {
		return nil, fmt.Errorf("invalid response content type: %s", resContentType)
	}

	var jsonRes json.RawMessage

	err = json.NewDecoder(res.Body).Decode(&jsonRes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return jsonRes, nil
}
