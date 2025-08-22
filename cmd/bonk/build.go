// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"log/slog"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"

	"github.com/spf13/cobra"
)

// buildCmd represents the build command.
var buildCmd = &cobra.Command{
	Use:   "build [paths...]",
	Short: "A brief description of your command",

	Args:       cobra.ArbitraryArgs,
	ArgAliases: []string{"paths"},

	Run: func(_ *cobra.Command, args []string) {
		slog.Info("Perforing Build")

		config := load.Config{}

		// If there's more than 1 arg, use the first as the root
		if len(args) > 0 {
			config.Dir = args[0]
			args[0] = "."
			slog.Debug("Using arg[0] as Dir", "dir", config.Dir)
		}

		cuectx := cuecontext.New()
		insts := load.Instances(args, &config)
		values, err := cuectx.BuildInstances(insts)
		if err != nil {
			slog.Error("Failed to build bonk project", "error", err.Error())
		}

		// Unify all of the values into a single source of truth
		value := cue.Value{}
		for _, valuePart := range values {
			value = value.Unify(valuePart)
		}

		holos := value.LookupPath(cue.MakePath(cue.Str("holos")))

		source := holos.Pos()
		slog.Debug("Source", "source", source.String())

		syn := holos.Syntax(
			cue.Final(),
			cue.Attributes(false),
		)

		str, err := format.Node(syn,
			format.TabIndent(false),
			format.Simplify(),
		)
		if err != nil {
			slog.Error("Failed to encode value", "error", err.Error())
		}
		slog.Info(string(str))
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
