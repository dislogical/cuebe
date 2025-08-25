// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	bonkv0 "go.bonk.build/api/go/proto/bonk/v0"
)

type streamHandler struct {
	slog.HandlerOptions

	sender grpc.ServerStreamingServer[bonkv0.StreamLogsResponse]
}

func (stream *streamHandler) Enabled(_ context.Context, level slog.Level) bool {
	return int(stream.Level.Level()) <= int(level)
}

func (stream *streamHandler) Handle(_ context.Context, record slog.Record) error {
	level := int32(record.Level)
	res := bonkv0.StreamLogsResponse_builder{
		Time:    timestamppb.New(record.Time),
		Message: &record.Message,
		Level:   &level,
		Attrs:   make(map[string]*structpb.Value, record.NumAttrs()),
	}

	// record.Attrs(func(attr slog.Attr) bool {
	// 	res.Attrs[attr.Key] = attr.Value
	// 	return true
	// })

	err := stream.sender.Send(res.Build())
	if err != nil {
		return fmt.Errorf("failed to send record across gRPC: %w", err)
	}

	return nil
}

func (stream *streamHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return stream
}

func (stream *streamHandler) WithGroup(name string) slog.Handler {
	return stream
}

func (s *grpcServer) StreamLogs(
	req *bonkv0.StreamLogsRequest,
	res grpc.ServerStreamingServer[bonkv0.StreamLogsResponse],
) error {
	slogDefault := slog.Default()

	slog.SetDefault(slog.New(
		&streamHandler{
			HandlerOptions: slog.HandlerOptions{
				Level:     slog.Level(req.GetLevel()),
				AddSource: req.GetAddSource(),
			},
			sender: res,
		},
	))

	// Sleep until the request is canceled
	<-res.Context().Done()

	slog.SetDefault(slogDefault)

	return nil
}
