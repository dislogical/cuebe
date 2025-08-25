// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk // import "go.bonk.build/api/go"

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/encoding/gocode/gocodec"

	"github.com/ValerySidorin/shclog"

	goplugin "github.com/hashicorp/go-plugin"

	bonkv0 "go.bonk.build/api/go/proto/bonk/v0"
)

var cuectx = cuecontext.New()

// The inputs passed to a task backend.
type TaskParams[Params any] struct {
	Params Params
	Inputs []string
	OutDir string
}

// Represents a backend capable of performing tasks.
type BonkBackend struct {
	Name         string
	Outputs      []string
	ParamsSchema cue.Value
	Exec         func(*slog.Logger, TaskParams[cue.Value]) error
}

// Factory to create a new task backend.
func NewBackend[Params any](
	name string,
	outputs []string,
	exec func(*slog.Logger, *TaskParams[Params]) error,
) BonkBackend {
	zero := new(Params)

	schema := cuectx.EncodeType(*zero)
	if schema.Err() != nil {
		panic(schema.Err())
	}

	return BonkBackend{
		Name:         name,
		Outputs:      outputs,
		ParamsSchema: schema,
		Exec: func(logger *slog.Logger, paramsCue TaskParams[cue.Value]) error {
			params := new(TaskParams[Params])
			params.Inputs = paramsCue.Inputs
			params.OutDir = paramsCue.OutDir
			err := paramsCue.Params.Decode(&params.Params)
			if err != nil {
				return fmt.Errorf("failed to decode task parameters: %w", err)
			}

			return exec(logger, params)
		},
	}
}

// Call from main() to start the plugin gRPC server.
func Serve(backends ...BonkBackend) {
	backendMap := make(map[string]BonkBackend, len(backends))
	for _, backend := range backends {
		backendMap[backend.Name] = backend
	}

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]goplugin.Plugin{
			PluginType: &bonkPluginServer{
				backends: backendMap,
			},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
		Logger:     shclog.New(slog.Default()),
	})
}

var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  0,
	MagicCookieKey:   "BONK_PLUGIN",
	MagicCookieValue: "backend",
}

const PluginType = "bonk"

// PRIVATE

type bonkPluginServer struct {
	goplugin.NetRPCUnsupportedPlugin
	goplugin.GRPCPlugin

	backends map[string]BonkBackend
}

func (p *bonkPluginServer) GRPCServer(_ *goplugin.GRPCBroker, s *grpc.Server) error {
	bonkv0.RegisterBonkPluginServiceServer(s, &grpcServer{
		decodeCodec: gocodec.New(cuectx, &gocodec.Config{}),
		backends:    p.backends,
	})

	return nil
}

// Here is the gRPC server that GRPCClient talks to.
type grpcServer struct {
	bonkv0.UnimplementedBonkPluginServiceServer

	decodeCodec *gocodec.Codec
	backends    map[string]BonkBackend
}

func (s *grpcServer) ConfigurePlugin(
	ctx context.Context,
	req *bonkv0.ConfigurePluginRequest,
) (*bonkv0.ConfigurePluginResponse, error) {
	respBuilder := bonkv0.ConfigurePluginResponse_builder{
		Features: []bonkv0.ConfigurePluginResponse_FeatureFlags{
			bonkv0.ConfigurePluginResponse_FEATURE_FLAGS_STREAMING_LOGGING,
		},
		Backends: make(map[string]*bonkv0.ConfigurePluginResponse_BackendDescription, len(s.backends)),
	}

	for name, backend := range s.backends {
		respBuilder.Backends[name] = bonkv0.ConfigurePluginResponse_BackendDescription_builder{
			Outputs: backend.Outputs,
		}.Build()
	}

	return respBuilder.Build(), nil
}

func (s *grpcServer) PerformTask(
	ctx context.Context,
	req *bonkv0.PerformTaskRequest,
) (*bonkv0.PerformTaskResponse, error) {
	backend, ok := s.backends[req.GetBackend()]
	if !ok {
		return nil, fmt.Errorf("backend %s is not registered to this plugin", req.GetBackend())
	}

	params := TaskParams[cue.Value]{
		Params: cue.Value{},
		Inputs: req.GetInputs(),
		OutDir: req.GetOutDirectory(),
	}

	err := s.decodeCodec.Validate(backend.ParamsSchema, req.GetParameters())
	if err != nil {
		return nil, fmt.Errorf(
			"params %s don't match required schema %s",
			req.GetParameters(),
			backend.ParamsSchema,
		)
	}

	params.Params, err = s.decodeCodec.Decode(req.GetParameters())
	if err != nil {
		return nil, fmt.Errorf("failed to decode parameters: %w", err)
	}

	err = backend.Exec(slog.Default().With("source", "task"), params)
	if err != nil {
		return nil, err
	}

	return bonkv0.PerformTaskResponse_builder{}.Build(), nil
}
