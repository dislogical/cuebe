// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main // import "go.bonk.build/plugins/k8s/kustomize"

import (
	"fmt"
	"os"
	"path"

	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	plugin "go.bonk.build/api/go"
)

const output = "kustomized.yaml"

type Params struct {
	Kustomization types.Kustomization `json:"-"`
}

func kustomize(p plugin.TaskParams[Params]) error {
	// Apply resources and any needed fixes
	p.Params.Kustomization.Resources = p.Inputs
	p.Params.Kustomization.FixKustomization()

	// Write out the kustomization.yaml file
	outFile, err := os.Create(path.Join(p.OutDir, konfig.DefaultKustomizationFileName()))
	if err != nil {
		return fmt.Errorf("failed to open kustomization file: %w", err)
	}

	enc := yaml.NewEncoder(outFile)

	err = enc.Encode(p.Params.Kustomization)
	if err != nil {
		return fmt.Errorf("failed to encode kustomization file as yaml: %w", err)
	}

	enc.Close()
	outFile.Close()

	// Perform the kustomization
	options := krusty.MakeDefaultOptions()
	options.LoadRestrictions = types.LoadRestrictionsNone
	kusty := krusty.MakeKustomizer(options)

	res, err := kusty.Run(filesys.MakeFsOnDisk(), p.OutDir)
	if err != nil {
		return fmt.Errorf("failed to perform kustomization: %w", err)
	}

	// Save the result
	resYaml, err := res.AsYaml()
	if err != nil {
		return fmt.Errorf("failed to encode kustomized content as yaml: %w", err)
	}

	err = os.WriteFile(path.Join(p.OutDir, output), resYaml, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to write kustomized content to file: %w", err)
	}

	return nil
}

func main() {
	plugin.Serve(
		plugin.NewBackend(
			"Kustomize",
			[]string{
				output,
			},
			kustomize,
		),
	)
}
