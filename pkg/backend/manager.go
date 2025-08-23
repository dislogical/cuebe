// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package backend // import "go.bonk.build/pkg/backend"

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	"go.bonk.build/pkg/task"
)

type BackendManager struct {
	cuectx   *cue.Context
	backends map[string]Backend
}

func NewBackendManager() *BackendManager {
	bm := &BackendManager{}
	bm.cuectx = cuecontext.New()
	bm.backends = make(map[string]Backend)

	return bm
}

func (bm *BackendManager) RegisterBackend(name string, impl Backend) error {
	_, ok := bm.backends[name]
	if ok {
		return fmt.Errorf("duplicate backend name: %s", name)
	}

	bm.backends[name] = impl

	return nil
}

func (bm *BackendManager) UnregisterBackend(name string) {
	delete(bm.backends, name)
}

func (bm *BackendManager) SendTask(tsk task.Task) error {
	backendName := tsk.Backend()

	backend, ok := bm.backends[backendName]
	if !ok {
		return fmt.Errorf("Backend %s not found", backendName)
	}

	outDir := tsk.GetOutputDirectory()
	stat, err := os.Stat(outDir)
	if err != nil || !stat.IsDir() {
		err := os.MkdirAll(outDir, 0o750)
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
	} else if tsk.CheckChecksum() {
		slog.Debug("checksums match, skipping task")

		return nil
	}

	err = backend.Execute(context.TODO(), bm.cuectx, tsk)
	if err != nil {
		return fmt.Errorf("failed to execute task: %w", err)
	}

	slog.Info("task succeeded, saving checksum")

	err = tsk.SaveChecksum()
	if err != nil {
		return fmt.Errorf("failed to checksum task: %w", err)
	}

	return nil
}

func (bm *BackendManager) Shutdown() {
	bm.backends = make(map[string]Backend)
}
