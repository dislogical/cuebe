// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"path"

	"cuelang.org/go/cue"
	"cuelang.org/go/pkg/encoding/yaml"

	plugin "github.com/dislogical/bonk/api/go"
)

var output = "resources.yaml"

type Params struct {
	Resources cue.Value `json:"resources" cue:"[...]"`
}

func genResources(p plugin.TaskParams[Params]) error {

	if len(p.Inputs) > 0 {
		return fmt.Errorf("resources task does not accept inputs")
	}

	resourcesYaml, err := yaml.MarshalStream(p.Params.Resources)
	if err != nil {
		return fmt.Errorf("failed to marshal resources into yaml: %w", err)
	}

	err = os.WriteFile(path.Join(p.OutDir, output), []byte(resourcesYaml), os.ModePerm)
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
