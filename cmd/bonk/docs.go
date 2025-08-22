// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

//go:build docs

package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// docsCmd represents the docs command
var docsCmd = &cobra.Command{
	Use:    "docs",
	Short:  "Generates documentation from the command tree",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		const docPath = "docs/cmd"
		cobra.CheckErr(os.MkdirAll(docPath, os.ModePerm))
		cobra.CheckErr(doc.GenMarkdownTree(rootCmd, docPath))
	},
}

func init() {
	rootCmd.AddCommand(docsCmd)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
