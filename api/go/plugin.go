// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin

import (
	"context"
	"fmt"
	"log/slog"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/encoding/gocode/gocodec"
	"google.golang.org/grpc"

	goplugin "github.com/hashicorp/go-plugin"

	protov1 "github.com/dislogical/cuebe/api/go/proto/cuebe/v1"
)

var cuectx = cuecontext.New()

type TaskParams[Params any] struct {
	Inputs []string
	Params *Params
	OutDir string
}

func NewBackend[Params any](name string, outputs []string, exec func(TaskParams[Params]) error) cuebeBackend {
	zero := new(Params)

	schema := cuectx.EncodeType(*zero)
	if schema.Err() != nil {
		panic(schema.Err())
	}

	slog.Info("backend schema", "backend", name, "schema", schema)

	return cuebeBackend{
		Name:    name,
		Outputs: outputs,
		Params:  schema,
		execThunk: func(paramsCue TaskParams[cue.Value]) error {
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

func Serve(backends ...cuebeBackend) {
	backendMap := make(map[string]cuebeBackend)
	for _, backend := range backends {
		backendMap[backend.Name] = backend
	}

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]goplugin.Plugin{
			PluginType: &cuebePluginServer{
				backends: backendMap,
			},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})
}

var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "CUEBE_PLUGIN",
	MagicCookieValue: "backend",
}

const PluginType = "cuebe"

// PRIVATE

type cuebeBackend struct {
	Name      string
	Outputs   []string
	Params    cue.Value
	execThunk func(TaskParams[cue.Value]) error
}

type cuebePluginServer struct {
	goplugin.NetRPCUnsupportedPlugin
	goplugin.GRPCPlugin

	backends map[string]cuebeBackend
}

func (p *cuebePluginServer) GRPCServer(broker *goplugin.GRPCBroker, s *grpc.Server) error {
	protov1.RegisterCuebePluginServiceServer(s, &grpcServer{
		decodeCodec: gocodec.New(cuectx, &gocodec.Config{}),
		backends:    p.backends,
	})
	return nil
}

// Here is the gRPC server that GRPCClient talks to.
type grpcServer struct {
	protov1.UnimplementedCuebePluginServiceServer

	decodeCodec *gocodec.Codec
	backends    map[string]cuebeBackend
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

	err := s.decodeCodec.Validate(backend.Params, req.GetParameters())
	if err != nil {
		return nil, fmt.Errorf("params %s don't match required schema %s", req.GetParameters(), backend.Params)
	}

	*params.Params, err = s.decodeCodec.Decode(req.GetParameters())
	if err != nil {
		return nil, fmt.Errorf("failed to decode parameters: %w", err)
	}

	err = backend.execThunk(params)
	if err != nil {
		return nil, err
	}

	return protov1.PerformTaskResponse_builder{}.Build(), nil
}
