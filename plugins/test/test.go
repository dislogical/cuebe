// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"log/slog"

	plugin "github.com/bonk-build/bonk/api/go"
)

type Params struct {
	Value int `json:"value"`
}

func main() {
	plugin.Serve(
		plugin.NewBackend(
			"Test",
			[]string{},
			func(param plugin.TaskParams[Params]) error {
				slog.Info("testing!", "param", param.Params)
				return nil
			},
		),
	)
}
