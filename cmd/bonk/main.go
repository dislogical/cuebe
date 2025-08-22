// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"log/slog"
	"os"
	"path"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.bonk.build/pkg/backend"
	"go.bonk.build/pkg/task"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "bonk",
	Short: "A cue-based configuration build system.",

	Run: func(cmd *cobra.Command, args []string) {
		cuectx := cuecontext.New()

		bem := backend.BackendManager{}
		defer bem.Shutdown()

		bem.Start(cmd.Context())

		cobra.CheckErr(
			bem.SendTask(task.New(
				"Test",
				"Test.Test",
				cuectx.CompileString(`value: 3`),
			)),
		)

		cobra.CheckErr(
			bem.SendTask(task.New(
				"Resources",
				"Test.Resources",
				cuectx.CompileString(`
				resources: [{
					apiVersion: "v1"
					kind: "Namespace"
					metadata: name: "Testing"
				}]`),
			)),
		)

		cwd, _ := os.Getwd()
		cobra.CheckErr(
			bem.SendTask(task.New(
				"Kustomize",
				"Test.Kustomize",
				cuectx.BuildExpr(ast.NewStruct()),
				path.Join(cwd, ".bonk/Test.Resources:Resources/resources.yaml"),
			)),
		)
	},
}

func init() {
	rootCmd.PersistentFlags().
		StringVarP(&cfgFile, "config", "c", "", "config file (default is .bonk.yaml)")

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in current directory with name ".bonk.yaml".
		viper.AddConfigPath(".")
		viper.SetConfigName(".bonk.yaml")
	}

	viper.AutomaticEnv()

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil {
		slog.Debug("Using config file", "file", viper.ConfigFileUsed())
	} else {
		slog.Debug("Not using config file", "error", err.Error())
	}
}

func main() {
	slog.SetDefault(
		slog.New(
			pterm.NewSlogHandler(
				pterm.DefaultLogger.
					WithWriter(rootCmd.OutOrStdout()).
					WithLevel(pterm.LogLevelDebug).
					WithTime(false),
			),
		),
	)

	err := rootCmd.Execute()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
