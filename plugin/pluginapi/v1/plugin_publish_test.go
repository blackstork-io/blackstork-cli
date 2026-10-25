// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package pluginapiv1

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

type publishCapturingClient struct {
	PluginServiceClient
	request *PublishRequest
}

func (c *publishCapturingClient) Publish(
	_ context.Context,
	req *PublishRequest,
	_ ...grpc.CallOption,
) (*PublishResponse, error) {
	c.request = req
	return &PublishResponse{}, nil
}

func TestClientPublishFuncEncodesContent(t *testing.T) {
	client := &publishCapturingClient{}
	transport := &grpcPlugin{
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	content := plugin.NewEmptySection(
		&plugin.BlockDetails{
			Kind:  "document",
			Name:  "report",
			ID:    "document",
			Depth: 0,
		},
		plugindata.Map{},
	)

	diags := transport.clientPublishFunc("test", client)(context.Background(), &plugin.PublishParams{
		DocumentName: "report",
		Config:       dataspec.NewBlock(nil, nil),
		Args:         dataspec.NewBlock(nil, nil),
		DataContext:  plugindata.Map{},
		Content:      content,
	})

	require.False(t, diags.HasErrors())
	require.NotNil(t, client.request)
	require.NotNil(t, client.request.Content)

	decoded, err := DecodeContent(client.request.Content)
	require.NoError(t, err)
	require.Equal(t, plugin.SectionKind, decoded.Kind())
	require.Equal(t, content.ID(), decoded.ID())
	require.Equal(t, content.BlockDetails(), decoded.BlockDetails())
}
