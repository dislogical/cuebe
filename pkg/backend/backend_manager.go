// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package backend

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/encoding/gocode/gocodec"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"

	goplugin "github.com/hashicorp/go-plugin"

	plugin "github.com/dislogical/bonk/api/go"
	protov1 "github.com/dislogical/bonk/api/go/proto/bonk/v1"
	"github.com/dislogical/bonk/pkg/task"
)

type Backend struct {
	client     protov1.BonkPluginServiceClient
	descriptor *protov1.ConfigurePluginResponse_BackendDescription
	outputs    []string
}

type BackendManager struct {
	cuectx   *cue.Context
	plugins  map[string]map[string]Backend
	backends map[string]Backend
}

func (bm *BackendManager) Start() {
	slog.Info("Starting Daemon")

	bm.cuectx = cuecontext.New()
	bm.plugins = make(map[string]map[string]Backend)
	bm.backends = make(map[string]Backend)

	for _, pluginPath := range []string{"./plugins/test", "./plugins/k8s/kustomize", "./plugins/k8s/resources"} {
		client := goplugin.NewClient(&goplugin.ClientConfig{
			HandshakeConfig: plugin.Handshake,
			Plugins: map[string]goplugin.Plugin{
				plugin.PluginType: &bonkPluginClient{},
			},
			Cmd:     exec.Command("go", "run", pluginPath),
			Managed: true,
			AllowedProtocols: []goplugin.Protocol{
				goplugin.ProtocolGRPC,
			},
		})

		rpc, err := client.Client()
		if err != nil {
			slog.Error("Failed to create client", "error", err)
			os.Exit(1)
		}

		bonkPlugin, err := rpc.Dispense(plugin.PluginType)
		if err != nil {
			slog.Error("Failed to dispense bonk plugin", "error", err)
			os.Exit(1)
		}

		bonkClient := bonkPlugin.(protov1.BonkPluginServiceClient)

		resp, err := bonkClient.ConfigurePlugin(context.TODO(), &protov1.ConfigurePluginRequest{})
		if err != nil {
			slog.Error("Failed to describe plugin backends", "error", err)
		}

		bm.plugins[pluginPath] = make(map[string]Backend)
		for name, backendDesc := range resp.GetBackends() {
			_, existed := bm.backends[name]
			if existed {
				slog.Warn("Duplicate backend detected", "name", name)
			}

			bm.plugins[pluginPath][name] = Backend{
				client:     bonkClient,
				descriptor: backendDesc,
				outputs:    backendDesc.GetOutputs(),
			}
			bm.backends[name] = bm.plugins[pluginPath][name]
		}
	}
}

func (bm *BackendManager) SendTask(t task.Task) error {
	backendName := t.Backend()
	backend, ok := bm.backends[backendName]
	if !ok {
		return fmt.Errorf("Backend %s not found", backendName)
	}

	outDir := t.GetOutputDirectory()

	if stat, err := os.Stat(outDir); err != nil || !stat.IsDir() {
		err := os.MkdirAll(outDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
	} else {
		if t.CheckChecksum() {
			slog.Debug("checksums match, skipping task")
			return nil
		}
	}

	taskReqBuilder := protov1.PerformTaskRequest_builder{
		Backend:      &backendName,
		Inputs:       t.Inputs,
		Parameters:   &structpb.Value{},
		OutDirectory: &outDir,
	}

	codec := gocodec.New(bm.cuectx, &gocodec.Config{})
	codec.Encode(t.Params, taskReqBuilder.Parameters)

	_, err := backend.client.PerformTask(context.TODO(), taskReqBuilder.Build())
	if err != nil {
		return fmt.Errorf("failed to perform task: %w", err)
	}

	slog.Info("task succeeded, saving checksum")
	err = t.SaveChecksum()
	if err != nil {
		return fmt.Errorf("failed to checksum task: %w", err)
	}

	return nil
}

func (bm *BackendManager) Shutdown() {
	bm.plugins = make(map[string]map[string]Backend)
	bm.backends = make(map[string]Backend)
	goplugin.CleanupClients()
}

// Plugin Client

type bonkPluginClient struct {
	goplugin.NetRPCUnsupportedPlugin
	goplugin.GRPCPlugin
}

func (p *bonkPluginClient) GRPCClient(ctx context.Context, broker *goplugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return protov1.NewBonkPluginServiceClient(c), nil
}
