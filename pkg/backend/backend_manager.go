// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package backend // import "go.bonk.build/pkg/backend"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/encoding/gocode/gocodec"

	goplugin "github.com/hashicorp/go-plugin"

	plugin "go.bonk.build/api/go"
	protov1 "go.bonk.build/api/go/proto/bonk/v1"
	"go.bonk.build/pkg/task"
)

type Backend struct {
	client     protov1.BonkPluginServiceClient
	descriptor *protov1.ConfigurePluginResponse_BackendDescription
	outputs    []string
}

type BackendManager struct {
	cancel   context.CancelCauseFunc
	cuectx   *cue.Context
	plugins  map[string]map[string]Backend
	backends map[string]Backend
}

func (bm *BackendManager) Start(ctx context.Context) {
	slog.Info("Starting Daemon")

	ctx, bm.cancel = context.WithCancelCause(ctx)
	bm.cuectx = cuecontext.New()
	bm.plugins = make(map[string]map[string]Backend)
	bm.backends = make(map[string]Backend)

	for _, pluginPath := range []string{"./plugins/test", "./plugins/k8s/kustomize", "./plugins/k8s/resources"} {
		client := goplugin.NewClient(&goplugin.ClientConfig{
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

		bonkClient, ok := bonkPlugin.(protov1.BonkPluginServiceClient)
		if !ok {
			slog.Error("got unexpected plugin client type")

			continue
		}

		configureCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		resp, err := bonkClient.ConfigurePlugin(configureCtx, &protov1.ConfigurePluginRequest{})
		cancel()
		if err != nil {
			slog.Error("Failed to describe plugin backends", "error", err)

			continue
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

func (bm *BackendManager) SendTask(tsk task.Task) error {
	backendName := tsk.Backend()

	backend, ok := bm.backends[backendName]
	if !ok {
		return fmt.Errorf("Backend %s not found", backendName)
	}

	outDir := tsk.GetOutputDirectory()

	stat, err := os.Stat(outDir)
	if err != nil || !stat.IsDir() {
		err := os.MkdirAll(outDir, 0o750)
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
	} else if tsk.CheckChecksum() {
		slog.Debug("checksums match, skipping task")

		return nil
	}

	taskReqBuilder := protov1.PerformTaskRequest_builder{
		Backend:      &backendName,
		Inputs:       tsk.Inputs,
		Parameters:   &structpb.Value{},
		OutDirectory: &outDir,
	}

	codec := gocodec.New(bm.cuectx, &gocodec.Config{})
	err = codec.Encode(tsk.Params, taskReqBuilder.Parameters)
	if err != nil {
		return fmt.Errorf("failed to encode parameters as protobuf: %w", err)
	}

	_, err = backend.client.PerformTask(context.TODO(), taskReqBuilder.Build())
	if err != nil {
		return fmt.Errorf("failed to perform task: %w", err)
	}

	slog.Info("task succeeded, saving checksum")

	err = tsk.SaveChecksum()
	if err != nil {
		return fmt.Errorf("failed to checksum task: %w", err)
	}

	return nil
}

func (bm *BackendManager) Shutdown() {
	bm.plugins = make(map[string]map[string]Backend)
	bm.backends = make(map[string]Backend)

	goplugin.CleanupClients()

	bm.cancel(errors.New("terminating"))
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
