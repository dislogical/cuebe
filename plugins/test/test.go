// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main // import "go.bonk.build/plugins/test"

import (
	plugin "go.bonk.build/api/go"
)

type Params struct {
	Value int `json:"value"`
}

func main() {
	plugin.Serve(
		plugin.NewBackend(
			"Test",
			[]string{},
			func(param *plugin.TaskParams[Params]) error {
				return nil
			},
		),
	)
}
