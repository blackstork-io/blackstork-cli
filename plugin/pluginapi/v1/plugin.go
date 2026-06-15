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
	"errors"
	"fmt"
	"log/slog"
	"time"

	goplugin "github.com/hashicorp/go-plugin"
	"github.com/hashicorp/hcl/v2"
	grpc "google.golang.org/grpc"

	"github.com/blackstork-io/blackstork-cli/pkg/diagnostics"
	"github.com/blackstork-io/blackstork-cli/plugin"
	"github.com/blackstork-io/blackstork-cli/plugin/plugindata"
)

var maxMsgSize = 1024 * 1024 * 100 // 100MB

var handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "PLUGINS_FOR",
	MagicCookieValue: "fabric",
}

type grpcPlugin struct {
	goplugin.Plugin
	log    *slog.Logger
	schema *plugin.Schema
}

func (p *grpcPlugin) GRPCServer(broker *goplugin.GRPCBroker, s *grpc.Server) error {
	RegisterPluginServiceServer(s, &grpcServer{
		schema: p.schema,
	})
	return nil
}

func (p *grpcPlugin) GRPCClient(
	ctx context.Context,
	broker *goplugin.GRPCBroker,
	c *grpc.ClientConn,
) (interface{}, error) {
	client := NewPluginServiceClient(c)
	res, err := client.GetSchema(ctx, &GetSchemaRequest{})
	if err != nil {
		return nil, err
	}
	schema, err := decodeSchema(res.Schema)
	if err != nil {
		return nil, err
	}
	for name, ds := range schema.DataSources {
		if ds == nil {
			return nil, fmt.Errorf("nil data source")
		}
		ds.DataFunc = p.clientDataFunc(name, client)
	}
	for name, cg := range schema.ContentProviders {
		if cg == nil {
			return nil, fmt.Errorf("nil content provider")
		}
		cg.ContentFunc = p.clientGenerateFunc(name, client)
	}
	for name, formatter := range schema.Formatters {
		if formatter == nil {
			return nil, fmt.Errorf("nil formatter")
		}
		formatter.FormatFunc = p.clientFormatFunc(name, client)
	}
	for name, pub := range schema.Publishers {
		if pub == nil {
			return nil, fmt.Errorf("nil publisher")
		}
		pub.PublishFunc = p.clientPublishFunc(name, client)
	}
	return schema, nil
}

func (p *grpcPlugin) callOptions() []grpc.CallOption {
	return []grpc.CallOption{
		grpc.MaxCallRecvMsgSize(maxMsgSize),
		grpc.MaxCallSendMsgSize(maxMsgSize),
	}
}

func (p *grpcPlugin) clientGenerateFunc(name string, client PluginServiceClient) plugin.ProvideContentFunc {
	return func(ctx context.Context, params *plugin.ProvideContentParams) (result *plugin.ContentProviderResult, err error) {
		p.log.DebugContext(ctx, "Calling a content provider", "name", name)
		defer func(start time.Time) {
			p.log.DebugContext(ctx, "Called a content provider", "name", name, "took", time.Since(start))
		}(time.Now())
		if params == nil {
			p.log.ErrorContext(ctx, "No parameters found for content provider")
			return nil, diagnostics.FromErr(errors.New("no parameters found for content provider"))
		}
		cfgEncoded, diag := encodeBlock(params.Config)
		if diag.HasErrors() {
			p.log.ErrorContext(ctx, "Error while encoding content block config", "err", diag.Error())
			return nil, diag
		}

		argsEncoded, diag := encodeBlock(params.Args)
		if diag.HasErrors() {
			p.log.ErrorContext(ctx, "Error while encoding content block args", "err", diag.Error())
			return nil, diag
		}

		data := encodeMapData(params.DataContext)
		res, err := client.ProvideContent(ctx, &ProvideContentRequest{
			Provider:    name,
			Config:      cfgEncoded,
			Args:        argsEncoded,
			DataContext: &data,
		}, p.callOptions()...)
		if err != nil {
			p.log.ErrorContext(ctx, "Error while generating content", "err", diag.Error())
			return nil, diagnostics.FromErr(err)
		}

		result, err = decodeContentProviderResult(res.GetResult())
		if err != nil {
			p.log.ErrorContext(ctx, "Error while decoding content generation result", "err", err)
			return nil, diagnostics.FromErr(err)
		}
		var diags diagnostics.Diag
		for _, diag := range decodeDiagnosticList(res.GetDiagnostics()) {

			if diag.Severity == hcl.DiagError {
				p.log.ErrorContext(ctx, "Error received from plugin", "err", diag.Error())
			} else {
				p.log.DebugContext(ctx, "Diagnostic received from plugin", "diag", diag)
			}
			diags.Append(diag)
		}
		return result, diags
	}
}

func (p *grpcPlugin) clientDataFunc(name string, client PluginServiceClient) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (data plugindata.Data, diags diagnostics.Diag) {
		p.log.DebugContext(ctx, "Calling a data source", "name", name)
		defer func(start time.Time) {
			p.log.DebugContext(ctx, "Called a data source", "name", name, "took", time.Since(start))
		}(time.Now())
		if params == nil {
			diags.Add("Data source error", "Nil params")
			return data, diags
		}
		cfgEncoded, diag := encodeBlock(params.Config)
		diags.Extend(diag)
		argsEncoded, diag := encodeBlock(params.Args)
		diags.Extend(diag)

		res, err := client.RetrieveData(ctx, &RetrieveDataRequest{
			Source: name,
			Config: cfgEncoded,
			Args:   argsEncoded,
		}, p.callOptions()...)
		if diags.AppendErr(err, "Failed to fetch data") {
			return data, diags
		}
		data = DecodeData(res.GetData())
		diags.Extend(decodeDiagnosticList(res.GetDiagnostics()))
		return data, diags
	}
}

func (p *grpcPlugin) clientFormatFunc(name string, client PluginServiceClient) plugin.FormatFunc {
	return func(ctx context.Context, params *plugin.FormatParams) (_ *plugin.FormattedContent, diags diagnostics.Diag) {
		p.log.DebugContext(ctx, "Calling a formatter", "name", name)
		defer func(start time.Time) {
			p.log.DebugContext(ctx, "Called a formatter", "name", name, "took", time.Since(start))
		}(time.Now())
		if params == nil {
			diags.Add("Formatter error", "Nil params")
			return
		}
		argsEncoded, diag := encodeBlock(params.Args)
		diags.Extend(diag)
		cfgEncoded, diag := encodeBlock(params.Config)
		diags.Extend(diag)

		datactx := encodeMapData(params.DataContext)
		content := encodeMapData(params.Content)

		res, err := client.FormatContent(ctx, &FormatContentRequest{
			Formatter:   name,
			Config:      cfgEncoded,
			Args:        argsEncoded,
			Content:     &content,
			DataContext: &datactx,
			Format:      params.Format,
		}, p.callOptions()...)

		if diags.AppendErr(err, "Failed to publish") {
			return nil, diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to publish",
				Detail:   err.Error(),
			}}
		}
		diags.Extend(decodeDiagnosticList(res.GetDiagnostics()))

		var result plugin.FormattedContent
		if res.Result != nil {
			result = plugin.FormattedContent{
				Content: res.Result.Content,
				Format:  res.Result.Format,
			}
		}

		return &result, diags
	}
}

func (p *grpcPlugin) clientPublishFunc(name string, client PluginServiceClient) plugin.PublishFunc {
	return func(ctx context.Context, params *plugin.PublishParams) (diags diagnostics.Diag) {
		p.log.DebugContext(ctx, "Calling publisher", "name", name)
		defer func(start time.Time) {
			p.log.DebugContext(ctx, "Called publisher", "name", name, "took", time.Since(start))
		}(time.Now())
		if params == nil {
			diags.Add("Publisher error", "Nil params")
			return diags
		}
		argsEncoded, diag := encodeBlock(params.Args)
		diags.Extend(diag)
		cfgEncoded, diag := encodeBlock(params.Config)
		diags.Extend(diag)
		datactx := encodeMapData(params.DataContext)

		var content *FormattedContent
		if params.FormattedContent != nil {
			content = &FormattedContent{
				Format:  params.FormattedContent.Format,
				Content: params.FormattedContent.Content,
			}
		}

		res, err := client.Publish(ctx, &PublishRequest{
			Publisher:        name,
			Config:           cfgEncoded,
			Args:             argsEncoded,
			DataContext:      &datactx,
			FormattedContent: content,
			DocumentName:     params.DocumentName,
		}, p.callOptions()...)

		if diags.AppendErr(err, "Failed to publish") {
			return diagnostics.Diag{{
				Severity: hcl.DiagError,
				Summary:  "Failed to publish",
				Detail:   err.Error(),
			}}
		}
		diags.Extend(decodeDiagnosticList(res.GetDiagnostics()))
		return diags
	}
}
