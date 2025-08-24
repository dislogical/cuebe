// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

//go:build docs

package main

import (
	"errors"
	"io"
	"os"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func trimFileNewlines(waitGroup *sync.WaitGroup, root *os.Root, file string) {
	stat, err := root.Stat(file)
	cobra.CheckErr(err)

	entryFile, err := root.OpenFile(file, os.O_RDWR, stat.Mode().Perm())
	cobra.CheckErr(err)

	size, err := entryFile.Seek(-1, io.SeekEnd)
	cobra.CheckErr(err)

	readBuf := make([]byte, 1)
	foundNewlines := int64(0)

	for {
		read, err := entryFile.Read(readBuf)
		cobra.CheckErr(err)
		if read != 1 {
			cobra.CheckErr(errors.New("failed to read byte in file"))
		}
		char := readBuf[0]

		if char == '\n' {
			_, err = entryFile.Seek(-2, io.SeekCurrent)
			cobra.CheckErr(err)
			foundNewlines++
		} else {
			break
		}
	}

	if foundNewlines > 1 {
		err = entryFile.Truncate(size + 1 - (foundNewlines - 1))
		cobra.CheckErr(err)
	}

	err = entryFile.Close()
	cobra.CheckErr(err)

	waitGroup.Done()
}

// docsCmd represents the docs command.
var docsCmd = &cobra.Command{
	Use:    "docs",
	Short:  "Generates documentation from the command tree",
	Hidden: true,
	Run: func(_ *cobra.Command, args []string) {
		const docPath = "docs/cmd"
		cobra.CheckErr(os.MkdirAll(docPath, 0o750))
		cobra.CheckErr(doc.GenMarkdownTree(rootCmd, docPath))

		waitGroup := sync.WaitGroup{}

		// Trim extra trailing newlines from each file
		root, err := os.OpenRoot(docPath)
		cobra.CheckErr(err)
		entries, err := os.ReadDir(docPath)
		cobra.CheckErr(err)
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			waitGroup.Add(1)
			go trimFileNewlines(&waitGroup, root, entry.Name())
		}

		waitGroup.Wait()
	},
}

func init() {
	rootCmd.AddCommand(docsCmd)
	rootCmd.DisableAutoGenTag = true
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
