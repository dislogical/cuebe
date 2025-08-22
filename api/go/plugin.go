// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin

//go:generate go tool -C ../.. buf generate

import (
	"context"
	"fmt"
	"log/slog"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/encoding/gocode/gocodec"
	"google.golang.org/grpc"

	goplugin "github.com/hashicorp/go-plugin"

	protov1 "github.com/bonk-build/bonk/api/go/proto/bonk/v1"
)

var cuectx = cuecontext.New()

// The inputs passed to a task backend
type TaskParams[Params any] struct {
	Inputs []string
	Params *Params
	OutDir string
}

// Represents a backend capable of performing tasks
type BonkBackend struct {
	Name         string
	Outputs      []string
	ParamsSchema cue.Value
	Exec         func(TaskParams[cue.Value]) error
}

// Factory to create a new task backend
func NewBackend[Params any](name string, outputs []string, exec func(TaskParams[Params]) error) BonkBackend {
	zero := new(Params)

	schema := cuectx.EncodeType(*zero)
	if schema.Err() != nil {
		panic(schema.Err())
	}

	slog.Info("backend schema", "backend", name, "schema", schema)

	return BonkBackend{
		Name:         name,
		Outputs:      outputs,
		ParamsSchema: schema,
		Exec: func(paramsCue TaskParams[cue.Value]) error {
			params := new(Params)
			paramsCue.Params.Decode(params)
			return exec(TaskParams[Params]{
				Inputs: paramsCue.Inputs,
				Params: params,
				OutDir: paramsCue.OutDir,
			})
		},
	}
}

// Call from main() to start the plugin gRPC server
func Serve(backends ...BonkBackend) {
	backendMap := make(map[string]BonkBackend)
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
	})
}

var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
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

func (p *bonkPluginServer) GRPCServer(broker *goplugin.GRPCBroker, s *grpc.Server) error {
	protov1.RegisterBonkPluginServiceServer(s, &grpcServer{
		decodeCodec: gocodec.New(cuectx, &gocodec.Config{}),
		backends:    p.backends,
	})
	return nil
}

// Here is the gRPC server that GRPCClient talks to.
type grpcServer struct {
	protov1.UnimplementedBonkPluginServiceServer

	decodeCodec *gocodec.Codec
	backends    map[string]BonkBackend
}

func (s *grpcServer) ConfigurePlugin(ctx context.Context, req *protov1.ConfigurePluginRequest) (*protov1.ConfigurePluginResponse, error) {
	respBuilder := protov1.ConfigurePluginResponse_builder{
		Backends: make(map[string]*protov1.ConfigurePluginResponse_BackendDescription, len(s.backends)),
	}

	for name, backend := range s.backends {
		respBuilder.Backends[name] = protov1.ConfigurePluginResponse_BackendDescription_builder{
			Outputs: backend.Outputs,
		}.Build()
	}

	return respBuilder.Build(), nil
}

func (s *grpcServer) PerformTask(ctx context.Context, req *protov1.PerformTaskRequest) (*protov1.PerformTaskResponse, error) {
	backend, ok := s.backends[req.GetBackend()]
	if !ok {
		return nil, fmt.Errorf("backend %s is not registered to this plugin", req.GetBackend())
	}

	params := TaskParams[cue.Value]{
		Inputs: req.GetInputs(),
		Params: &cue.Value{},
		OutDir: req.GetOutDirectory(),
	}

	err := s.decodeCodec.Validate(backend.ParamsSchema, req.GetParameters())
	if err != nil {
		return nil, fmt.Errorf("params %s don't match required schema %s", req.GetParameters(), backend.ParamsSchema)
	}

	*params.Params, err = s.decodeCodec.Decode(req.GetParameters())
	if err != nil {
		return nil, fmt.Errorf("failed to decode parameters: %w", err)
	}

	err = backend.Exec(params)
	if err != nil {
		return nil, err
	}

	return protov1.PerformTaskResponse_builder{}.Build(), nil
}
