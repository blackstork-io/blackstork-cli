package pluginapiv1

import (
	context "context"
	"fmt"
	"log/slog"
	"time"

	goplugin "github.com/hashicorp/go-plugin"
	"github.com/hashicorp/hcl/v2"
	grpc "google.golang.org/grpc"

	"github.com/blackstork-io/fabric/pkg/diagnostics"
	"github.com/blackstork-io/fabric/plugin"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

var maxMsgSize = 1024 * 1024 * 100 // 100MB

var handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "PLUGINS_FOR",
	MagicCookieValue: "fabric",
}

type grpcPlugin struct {
	goplugin.Plugin
	logger *slog.Logger
	schema *plugin.Schema
}

func (p *grpcPlugin) GRPCServer(broker *goplugin.GRPCBroker, s *grpc.Server) error {
	RegisterPluginServiceServer(s, &grpcServer{
		schema: p.schema,
	})
	return nil
}

func (p *grpcPlugin) GRPCClient(ctx context.Context, broker *goplugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
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
	return func(ctx context.Context, params *plugin.ProvideContentParams) (result *plugin.ContentResult, diags diagnostics.Diag) {
		p.logger.DebugContext(ctx, "Calling a content provider", "name", name)
		defer func(start time.Time) {
			p.logger.DebugContext(ctx, "Called a content provider", "name", name, "took", time.Since(start))
		}(time.Now())
		if params == nil {
			diags.Add("Content provider error", "Nil params")
			return
		}
		cfgEncoded, diag := encodeBlock(params.Config)
		diags.Extend(diag)
		argsEncoded, diag := encodeBlock(params.Args)
		diags.Extend(diag)
		if diags.HasErrors() {
			return
		}
		res, err := client.ProvideContent(ctx, &ProvideContentRequest{
			Provider:    name,
			Config:      cfgEncoded,
			Args:        argsEncoded,
			DataContext: encodeMapData(params.DataContext),
			ContentId:   params.ContentID,
		}, p.callOptions()...)
		if diags.AppendErr(err, "Failed to generate content") {
			return
		}
		result = decodeContentResult(res.GetResult())
		diags.Extend(decodeDiagnosticList(res.GetDiagnostics()))
		return result, diags
	}
}

func (p *grpcPlugin) clientDataFunc(name string, client PluginServiceClient) plugin.RetrieveDataFunc {
	return func(ctx context.Context, params *plugin.RetrieveDataParams) (data plugindata.Data, diags diagnostics.Diag) {
		p.logger.DebugContext(ctx, "Calling a data source", "name", name)
		defer func(start time.Time) {
			p.logger.DebugContext(ctx, "Called a data source", "name", name, "took", time.Since(start))
		}(time.Now())
		if params == nil {
			diags.Add("Data source error", "Nil params")
			return
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
			return
		}
		data = decodeData(res.GetData())
		diags.Extend(decodeDiagnosticList(res.GetDiagnostics()))
		return
	}
}

func (p *grpcPlugin) clientFormatFunc(name string, client PluginServiceClient) plugin.FormatFunc {
	return func(ctx context.Context, params *plugin.FormatParams) (_ *plugin.FormattedContent, diags diagnostics.Diag) {
		p.logger.DebugContext(ctx, "Calling a formatter", "name", name)
		defer func(start time.Time) {
			p.logger.DebugContext(ctx, "Called a formatter", "name", name, "took", time.Since(start))
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
			Formatter:    name,
			Config:       cfgEncoded,
			Args:         argsEncoded,
			Content:      content,
			DataContext:  datactx,
			Format:       params.Format,
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
				Format: res.Result.Format,
			}
		}

		return &result, diags
	}
}

func (p *grpcPlugin) clientPublishFunc(name string, client PluginServiceClient) plugin.PublishFunc {
	return func(ctx context.Context, params *plugin.PublishParams) (diags diagnostics.Diag) {
		p.logger.DebugContext(ctx, "Calling publisher", "name", name)
		defer func(start time.Time) {
			p.logger.DebugContext(ctx, "Called publisher", "name", name, "took", time.Since(start))
		}(time.Now())
		if params == nil {
			diags.Add("Publisher error", "Nil params")
			return
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
			DataContext:      datactx,
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
		return
	}
}
