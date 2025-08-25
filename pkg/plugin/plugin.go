// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"google.golang.org/grpc"

	goplugin "github.com/hashicorp/go-plugin"

	bonkv0 "go.bonk.build/api/go/proto/bonk/v0"
)

type Plugin struct {
	client   bonkv0.BonkPluginServiceClient
	backends map[string]PluginBackend
}

func NewPlugin(ctx context.Context, client bonkv0.BonkPluginServiceClient) (*Plugin, error) {
	configureCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	resp, err := client.ConfigurePlugin(configureCtx, &bonkv0.ConfigurePluginRequest{})
	cancel()
	if err != nil {
		return nil, fmt.Errorf("failed to describe plugin: %w", err)
	}

	plugin := &Plugin{
		client:   client,
		backends: make(map[string]PluginBackend, len(resp.GetBackends())),
	}

	for _, feature := range resp.GetFeatures() {
		switch feature {
		default:
			// unsupported feature, ignore

		case bonkv0.ConfigurePluginResponse_FEATURE_FLAGS_STREAMING_LOGGING:
			err = plugin.HandleFeatureLogging(ctx)
			if err != nil {
				slog.WarnContext(ctx, "failed to configure streaming logging for plugin", "error", err)
			}
		}
	}

	for name, backendDesc := range resp.GetBackends() {
		_, existed := plugin.backends[name]
		if existed {
			slog.WarnContext(ctx, "duplicate backend detected", "name", name)
		}

		plugin.backends[name] = PluginBackend{
			name:       name,
			plugin:     plugin,
			descriptor: backendDesc,
		}
	}

	return plugin, nil
}

func (p *Plugin) HandleFeatureLogging(ctx context.Context) error {
	defaultLevel := int32(slog.LevelInfo)
	addSource := false

	req := bonkv0.StreamLogsRequest_builder{
		Level:     &defaultLevel,
		AddSource: &addSource,
	}

	logStream, err := p.client.StreamLogs(ctx, req.Build())
	if err != nil {
		return fmt.Errorf("failed to call client.StreamLogs: %w", err)
	}
	go func() {
		recvCtx := logStream.Context()
		for {
			msg, err := logStream.Recv()
			if err != nil {
				if recvCtx.Err() != nil || errors.Is(err, io.EOF) {
					// If this occurs, the log stream is imply shutting down and we should exit
					break
				} else {
					slog.ErrorContext(recvCtx, "received error on log stream", "error", err, "context err", recvCtx.Err())

					continue
				}
			}

			record := slog.NewRecord(
				msg.GetTime().AsTime(),
				slog.Level(msg.GetLevel()),
				msg.GetMessage(),
				0,
			)

			record.Add("source", "plugin")

			// for key, value := range msg.GetAttrs() {
			// 	record.AddAttrs(slog.Attr{
			// 		Key:   key,
			// 		Value: value,
			// 	})
			// }

			slogHandler := slog.Default().Handler()
			if slogHandler.Enabled(recvCtx, record.Level) {
				err = slogHandler.Handle(recvCtx, record)
				if err != nil {
					slog.ErrorContext(recvCtx, "failed to handle message")
				}
			}
		}
	}()

	return nil
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
	return bonkv0.NewBonkPluginServiceClient(c), nil
}
