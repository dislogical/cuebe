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
	"go.bonk.build/pkg/plugin"
	"go.bonk.build/pkg/scheduler"
	"go.bonk.build/pkg/task"
)

var (
	cfgFile     string
	concurrency uint
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "bonk",
	Short: "A cue-based configuration build system.",

	Run: func(cmd *cobra.Command, args []string) {
		cuectx := cuecontext.New()

		bem := backend.NewBackendManager()
		defer bem.Shutdown()

		pum := plugin.NewPluginManager(bem)
		defer pum.Shutdown()

		sched := scheduler.NewScheduler(bem, concurrency)
		defer sched.Run()

		plugins := []string{
			"go.bonk.build/plugins/test",
			"go.bonk.build/plugins/k8s/resources",
			"go.bonk.build/plugins/k8s/kustomize",
		}

		for _, pluginPath := range plugins {
			cobra.CheckErr(pum.StartPlugin(cmd.Context(), pluginPath))
		}

		cobra.CheckErr(
			sched.AddTask(
				task.New(
					"test:Test",
					"Test.Test",
					cuectx.CompileString(`value: 3`),
				),
			),
		)

		cobra.CheckErr(
			sched.AddTask(
				task.New(
					"resources:Resources",
					"Test.Resources",
					cuectx.CompileString(`
					resources: [{
						apiVersion: "v1"
						kind: "Namespace"
						metadata: name: "Testing"
					}]`),
				),
			),
		)

		cwd, _ := os.Getwd()
		cobra.CheckErr(
			sched.AddTask(
				task.New(
					"kustomize:Kustomize",
					"Test.Kustomize",
					cuectx.BuildExpr(ast.NewStruct()),
					path.Join(cwd, ".bonk/Test.Resources:resources:Resources/resources.yaml"),
				),
				"Test.Resources:resources:Resources",
			),
		)
	},
}

func init() {
	rootCmd.PersistentFlags().
		StringVarP(&cfgFile, "config", "c", "", "config file (default is .bonk.yaml)")
	rootCmd.PersistentFlags().
		UintVarP(&concurrency, "concurrency", "j", 100, "The number of goroutines to run")

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
		slog.Debug("using config file", "file", viper.ConfigFileUsed())
	} else {
		slog.Debug("not using config file", "error", err.Error())
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
