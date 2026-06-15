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
	context "context"
	"log/slog"

	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"

	"github.com/blackstork-io/blackstork-cli/plugin"
)

func Serve(schema *plugin.Schema) {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: handshake,
		Plugins: map[string]goplugin.Plugin{
			schema.Name: &grpcPlugin{schema: schema},
		},
		GRPCServer: func(opts []grpc.ServerOption) *grpc.Server {
			opts = append(opts, grpc.MaxRecvMsgSize(maxMsgSize))
			return grpc.NewServer(opts...)
		},
		Logger: loggerForGoplugin(),
	})
}

type grpcServer struct {
	schema *plugin.Schema
	UnimplementedPluginServiceServer
}

func (srv *grpcServer) GetSchema(ctx context.Context, req *GetSchemaRequest) (*GetSchemaResponse, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(r)
		}
	}()
	schema, diags := encodeSchema(srv.schema)
	if diags.HasErrors() {
		return nil, status.Errorf(codes.Internal, "failed to encode schema: %v", diags)
	}
	return &GetSchemaResponse{Schema: schema}, nil
}

func (srv *grpcServer) RetrieveData(ctx context.Context, req *RetrieveDataRequest) (*RetrieveDataResponse, error) {
	slog.DebugContext(ctx, "RetrieveData")
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "RetrieveData done", "panic", r)
			panic(r)
		} else {
			slog.DebugContext(ctx, "RetrieveData done")
		}
	}()
	source := req.GetSource()
	cfg, err := decodeBlock(req.GetConfig())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to decode config: %v", err)
	}
	args, err := decodeBlock(req.GetArgs())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to decode args: %v", err)
	}
	data, diags := srv.schema.RetrieveData(ctx, source, &plugin.RetrieveDataParams{
		Config: cfg,
		Args:   args,
	})
	return &RetrieveDataResponse{
		Data:        EncodeData(data),
		Diagnostics: encodeDiagnosticList(diags),
	}, nil
}

func (srv *grpcServer) ProvideContent(
	ctx context.Context,
	req *ProvideContentRequest,
) (*ProvideContentResponse, error) {
	slog.DebugContext(ctx, "ProvideContent")
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "ProvideContent done", "panic", r)
			panic(r)
		} else {
			slog.DebugContext(ctx, "ProvideContent done")
		}
	}()
	provider := req.GetProvider()
	cfg, err := decodeBlock(req.GetConfig())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to decode config: %v", err)
	}
	args, err := decodeBlock(req.GetArgs())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to decode args: %v", err)
	}
	datactx := decodeMapData(req.GetDataContext().GetValue())
	result, err := srv.schema.ProvideContent(ctx, provider, &plugin.ProvideContentParams{
		Config:      cfg,
		Args:        args,
		DataContext: datactx,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error while producing content: %v", err)
	}
	return &ProvideContentResponse{
		Result: encodeContentProviderResult(result),
		// FIXME: receiveing no diagnostics from the plugin
		Diagnostics: []*Diagnostic{},
	}, nil
}

func (srv *grpcServer) FormatContent(ctx context.Context, req *FormatContentRequest) (*FormatContentResponse, error) {
	slog.DebugContext(ctx, "Formatting")
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "Formatting failed", "panic", r)
			panic(r)
		} else {
			slog.DebugContext(ctx, "Formatter done")
		}
	}()
	formatterName := req.GetFormatter()
	cfg, err := decodeBlock(req.GetConfig())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to decode config: %v", err)
	}
	args, err := decodeBlock(req.GetArgs())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to decode args: %v", err)
	}
	datactx := decodeMapData(req.GetDataContext().GetValue())
	contentTree := decodeMapData(req.GetContent().GetValue())
	content, diags := srv.schema.Format(ctx, formatterName, &plugin.FormatParams{
		Config:      cfg,
		Args:        args,
		Content:     contentTree,
		DataContext: datactx,
	})
	return &FormatContentResponse{
		Result: &FormattedContent{
			Content: content.Content,
			Format:  content.Format,
		},
		Diagnostics: encodeDiagnosticList(diags),
	}, nil
}

func (srv *grpcServer) Publish(ctx context.Context, req *PublishRequest) (*PublishResponse, error) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "Publishing failed", "panic", r)
			panic(r)
		} else {
			slog.DebugContext(ctx, "Publish done")
		}
	}()
	publisher := req.GetPublisher()
	cfg, err := decodeBlock(req.GetConfig())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to decode config: %v", err)
	}
	args, err := decodeBlock(req.GetArgs())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to decode args: %v", err)
	}
	datactx := decodeMapData(req.GetDataContext().GetValue())
	contentRaw := req.GetContent()
	var content plugin.Content
	if contentRaw != nil {
		content, err = DecodeContent(contentRaw)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to decode content: %v", err)
		}
	}
	formattedContent := decodeFormattedContent(req.GetFormattedContent())
	diags := srv.schema.Publish(ctx, publisher, &plugin.PublishParams{
		Config:           cfg,
		Args:             args,
		DataContext:      datactx,
		DocumentName:     req.GetDocumentName(),
		Content:          content,
		FormattedContent: formattedContent,
	})
	return &PublishResponse{
		Diagnostics: encodeDiagnosticList(diags),
	}, nil
}
