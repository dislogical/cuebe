// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin // import "go.bonk.build/pkg/plugin"

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"cuelang.org/go/cue"

	protov1 "go.bonk.build/api/go/proto/bonk/v1"
	"go.bonk.build/pkg/task"
)

type PluginBackend struct {
	plugin     *Plugin
	name       string
	descriptor *protov1.ConfigurePluginResponse_BackendDescription
}

func (pb *PluginBackend) Outputs() []string {
	return pb.descriptor.GetOutputs()
}

func (pb *PluginBackend) Execute(ctx context.Context, cuectx *cue.Context, tsk task.Task) error {
	outDir := tsk.GetOutputDirectory()
	taskReqBuilder := protov1.PerformTaskRequest_builder{
		Backend:      &pb.name,
		Inputs:       tsk.Inputs,
		Parameters:   &structpb.Struct{},
		OutDirectory: &outDir,
	}

	err := tsk.Params.Decode(taskReqBuilder.Parameters)
	if err != nil {
		return fmt.Errorf("failed to encode parameters as protobuf: %w", err)
	}

	_, err = pb.plugin.client.PerformTask(ctx, taskReqBuilder.Build())
	if err != nil {
		return fmt.Errorf("failed to call perform task: %w", err)
	}

	err = tsk.SaveChecksum()
	if err != nil {
		return fmt.Errorf("failed to checksum task: %w", err)
	}

	return nil
}
