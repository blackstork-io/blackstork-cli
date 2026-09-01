// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

// Package pluginapiv1 implements version 1 of the BlackStork plugin protocol.
package pluginapiv1

import (
	"fmt"
	"log/slog"
	"os/exec"

	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/blackstork-io/blackstork-cli/pkg/utils/sloghclog"
	"github.com/blackstork-io/blackstork-cli/plugin"
)

func NewClient(name, binaryPath string, log *slog.Logger) (a *plugin.Schema, closefn func() error, err error) {
	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: handshake,
		Plugins: map[string]goplugin.Plugin{
			name: &grpcPlugin{
				log: log,
			},
		},
		Cmd: exec.Command(binaryPath),
		AllowedProtocols: []goplugin.Protocol{
			goplugin.ProtocolGRPC,
		},
		Logger: sloghclog.Adapt(
			log,
			sloghclog.Name("plugin."+name),
			// disable code location reporting, it's always going to be incorrect for remote plugin logs
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
		_ = rpcClient.Close() // Best-effort cleanup after a failed handshake.
		return nil, nil, fmt.Errorf("failed to dispense plugin: %w", err)
	}
	plg, ok := raw.(*plugin.Schema)
	if !ok {
		_ = rpcClient.Close() // Best-effort cleanup after receiving an unexpected plugin type.
		return nil, nil, fmt.Errorf("unexpected plugin type: %T", raw)
	}
	return plg, rpcClient.Close, nil
}
