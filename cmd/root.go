// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package cmd

import (
	"log/slog"
	"os"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var logger *slog.Logger

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cuebe",
	Short: "A cue-based configuration build system.",
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is .cuebe.yaml)")

	logger = slog.New(
		pterm.NewSlogHandler(
			pterm.DefaultLogger.
				WithWriter(rootCmd.OutOrStdout()).
				WithLevel(pterm.LogLevelDebug).
				WithTime(false)))

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in current directory with name ".cuebe.yaml".
		viper.AddConfigPath(".")
		viper.SetConfigName(".cuebe.yaml")
	}

	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		logger.Debug("Using config file", "file", viper.ConfigFileUsed())
	} else {
		logger.Debug("Not using config file", "error", err.Error())
	}
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
