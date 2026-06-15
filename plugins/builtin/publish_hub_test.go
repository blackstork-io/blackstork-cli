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
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/proto"

	client_mocks "github.com/blackstork-io/blackstork-cli/mocks/plugins/builtin/hubapi"
	"github.com/blackstork-io/blackstork-cli/plugin"
	pluginapiv1 "github.com/blackstork-io/blackstork-cli/plugin/pluginapi/v1"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/plugin/plugintest"
	"github.com/blackstork-io/blackstork-cli/plugins/builtin/hubapi"
)

type HubPublisherTestSuite struct {
	suite.Suite

	publisher      *plugin.Publisher
	ctx            context.Context
	cli            *client_mocks.Client
	storedApiURL   string
	storedApiToken string
}

func TestHubPublisherTestSuite(t *testing.T) {
	suite.Run(t, new(HubPublisherTestSuite))
}

func (s *HubPublisherTestSuite) SetupSuite() {
	s.publisher = makeHubPublisher("v0.0.0", func(apiURL, apiToken, version string) hubapi.Client {
		s.storedApiToken = apiToken
		s.storedApiURL = apiURL
		s.Equal("v0.0.0", version)
		return s.cli
	}, nil, nil)
	s.ctx = context.Background()
}

func (s *HubPublisherTestSuite) SetupTest() {
	s.cli = &client_mocks.Client{}
}

func (s *HubPublisherTestSuite) TearDownTest() {
	s.cli.AssertExpectations(s.T())
}

func (s *HubPublisherTestSuite) TestPublish() {
	datactx := plugindata.Map{
		"document": plugindata.Map{
			"meta": plugindata.Map{
				"name": plugindata.String("test_document"),
			},
			"content": plugindata.Map{
				"type": plugindata.String("section"),
				"children": plugindata.List{
					plugindata.Map{
						"type":     plugindata.String("element"),
						"markdown": plugindata.String("# Header 1"),
						"meta": plugindata.Map{
							"provider": plugindata.String("title"),
							"plugin":   plugindata.String("blackstork/builtin"),
						},
					},
					plugindata.Map{
						"type":     plugindata.String("element"),
						"markdown": plugindata.String("Lorem ipsum dolor sit amet, consectetur adipiscing elit."),
					},
				},
			},
		},
	}
	ts := time.Now()
	s.cli.On("CreateDocument", mock.Anything, &hubapi.DocumentParams{
		Title: "Test Title",
	}).Return(&hubapi.Document{
		ID:        "document_001",
		Title:     "Test Title",
		ContentID: nil,
		CreatedAt: ts,
		UpdatedAt: ts,
	}, nil)
	s.cli.On("UploadDocumentContent", mock.Anything, "document_001", mock.MatchedBy(func(got *pluginapiv1.Content) bool {
		doc, _ := parseScope(datactx)
		return proto.Equal(got, pluginapiv1.EncodeContent(doc))
	})).Return(&hubapi.DocumentContent{
		ID:        "content_001",
		CreatedAt: ts,
	}, nil)
	diags := s.publisher.PublishFunc(s.ctx, &plugin.PublishParams{
		Config: plugintest.NewTestDecoder(s.T(), s.publisher.Config).
			SetAttr("api_url", cty.StringVal("test-url")).
			SetAttr("api_token", cty.StringVal("test-token")).
			Decode(),
		Args: plugintest.NewTestDecoder(s.T(), s.publisher.Args).
			SetAttr("title", cty.StringVal("Test Title")).
			Decode(),
		DataContext: datactx,
	})
	s.Equal("test-url", s.storedApiURL)
	s.Equal("test-token", s.storedApiToken)
	s.Len(diags, 0)
}
