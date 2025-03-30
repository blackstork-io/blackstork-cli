package pluginapiv1

import (
	"fmt"
	"log/slog"
	"os/exec"

	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/blackstork-io/fabric/pkg/sloghclog"
	"github.com/blackstork-io/fabric/plugin"
)

func NewClient(name, binaryPath string, logger *slog.Logger) (p *plugin.Schema, closefn func() error, err error) {
	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: handshake,
		Plugins: map[string]goplugin.Plugin{
			name: &grpcPlugin{
				logger: logger,
			},
		},
		Cmd: exec.Command(binaryPath),
		AllowedProtocols: []goplugin.Protocol{
			goplugin.ProtocolGRPC,
		},
		Logger: sloghclog.Adapt(
			logger,
			sloghclog.Name("plugin."+name),
			// disable code location reporting, it's always going to be incorrect
			// for remote plugin logs
			sloghclog.AddSource(false),
		),
		GRPCDialOptions: []grpc.DialOption{
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(maxMsgSize),
				grpc.MaxCallSendMsgSize(maxMsgSize),
			),
		},
	})
	rpcClient, err := client.Client()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create plugin client: %w", err)
	}
	raw, err := rpcClient.Dispense(name)
	if err != nil {
		closeErr := rpcClient.Close()
		if closeErr != nil {
			logger.Warn("failed to close RPC client after dispense error", "err", closeErr)
		}
		return nil, nil, fmt.Errorf("failed to dispense plugin: %w", err)
	}
	plg, ok := raw.(*plugin.Schema)
	if !ok {
		closeErr := rpcClient.Close()
		if closeErr != nil {
			logger.Warn("failed to close RPC client after type assertion error", "err", closeErr)
		}
		return nil, nil, fmt.Errorf("unexpected plugin type: %T", raw)
	}
	return plg, rpcClient.Close, nil
}
