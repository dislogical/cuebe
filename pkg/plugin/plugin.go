// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"

	goplugin "github.com/hashicorp/go-plugin"

	protov1 "go.bonk.build/api/go/proto/bonk/v1"
)

type Plugin struct {
	client   protov1.BonkPluginServiceClient
	backends map[string]PluginBackend
}

func NewPlugin(ctx context.Context, client protov1.BonkPluginServiceClient) (*Plugin, error) {
	configureCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	resp, err := client.ConfigurePlugin(configureCtx, &protov1.ConfigurePluginRequest{})
	cancel()
	if err != nil {
		return nil, fmt.Errorf("failed to describe plugin: %w", err)
	}

	plugin := &Plugin{
		client:   client,
		backends: make(map[string]PluginBackend, len(resp.GetBackends())),
	}

	for name, backendDesc := range resp.GetBackends() {
		_, existed := plugin.backends[name]
		if existed {
			slog.Warn("duplicate backend detected", "name", name)
		}

		plugin.backends[name] = PluginBackend{
			name:       name,
			plugin:     plugin,
			descriptor: backendDesc,
		}
	}

	return plugin, nil
}

// Plugin Client

type bonkPluginClient struct {
	goplugin.NetRPCUnsupportedPlugin
	goplugin.GRPCPlugin
}

func (p *bonkPluginClient) GRPCClient(
	_ context.Context,
	_ *goplugin.GRPCBroker,
	c *grpc.ClientConn,
) (any, error) {
	return protov1.NewBonkPluginServiceClient(c), nil
}
