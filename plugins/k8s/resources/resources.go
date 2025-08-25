// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main // import "go.bonk.build/plugins/k8s/resources"

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"

	"cuelang.org/go/cue"
	"cuelang.org/go/pkg/encoding/yaml"

	plugin "go.bonk.build/api/go"
)

const output = "resources.yaml"

type Params struct {
	Resources cue.Value `cue:"[...]" json:"resources"`
}

func genResources(_ *slog.Logger, params *plugin.TaskParams[Params]) error {
	if len(params.Inputs) > 0 {
		return errors.New("resources task does not accept inputs")
	}

	resourcesYaml, err := yaml.MarshalStream(params.Params.Resources)
	if err != nil {
		return fmt.Errorf("failed to marshal resources into yaml: %w", err)
	}

	err = os.WriteFile(path.Join(params.OutDir, output), []byte(resourcesYaml), 0o600)
	if err != nil {
		return fmt.Errorf("failed to write resources yaml to disk: %w", err)
	}

	return nil
}

func main() {
	plugin.Serve(
		plugin.NewBackend(
			"Resources",
			[]string{
				output,
			},
			genResources,
		),
	)
}
