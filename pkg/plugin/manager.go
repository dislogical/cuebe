// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin // import "go.bonk.build/pkg/plugin"

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path"

	goplugin "github.com/hashicorp/go-plugin"

	plugin "go.bonk.build/api/go"
	protov1 "go.bonk.build/api/go/proto/bonk/v1"
	"go.bonk.build/pkg/backend"
)

type PluginManager struct {
	plugins map[string]*Plugin

	backend *backend.BackendManager
}

func NewPluginManager(backend *backend.BackendManager) *PluginManager {
	pm := &PluginManager{}
	pm.plugins = make(map[string]*Plugin)
	pm.backend = backend

	return pm
}

func (pm *PluginManager) StartPlugin(ctx context.Context, pluginPath string) error {
	pluginName := path.Base(pluginPath)

	process := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: plugin.Handshake,
		Plugins: map[string]goplugin.Plugin{
			plugin.PluginType: &bonkPluginClient{},
		},
		Cmd:     exec.CommandContext(ctx, "go", "run", pluginPath),
		Managed: true,
		AllowedProtocols: []goplugin.Protocol{
			goplugin.ProtocolGRPC,
		},
	})

	rpcClient, err := process.Client()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	pluginClient, err := rpcClient.Dispense(plugin.PluginType)
	if err != nil {
		return fmt.Errorf("failed to dispense bonk plugin: %w", err)
	}

	bonkClient, ok := pluginClient.(protov1.BonkPluginServiceClient)
	if !ok {
		return errors.New("got unexpected plugin client type")
	}

	plug, err := NewPlugin(ctx, bonkClient)
	if err != nil {
		return fmt.Errorf("failed to create plugin %s: %w", pluginName, err)
	}

	pm.plugins[pluginName] = plug

	for backendName, backend := range plug.backends {
		err = pm.backend.RegisterBackend(fmt.Sprintf("%s:%s", pluginName, backendName), &backend)
		if err != nil {
			return fmt.Errorf("failed to register plugin %s backend %s: %w", pluginName, backendName, err)
		}
	}

	return nil
}

func (pm *PluginManager) Shutdown() {
	for pluginName, plugin := range pm.plugins {
		for backendName := range plugin.backends {
			pm.backend.UnregisterBackend(fmt.Sprintf("%s:%s", pluginName, backendName))
		}
	}
	pm.plugins = make(map[string]*Plugin)

	goplugin.CleanupClients()
}
