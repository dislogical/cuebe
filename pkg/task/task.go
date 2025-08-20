// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"reflect"

	"cuelang.org/go/cue"
)

type TaskId struct {
	id      string
	backend string
}

func (id *TaskId) String() string {
	return fmt.Sprintf("%s:%s", id.id, id.backend)
}

func (id *TaskId) GetOutputDirectory() string {
	return path.Join(".cuebe", id.String())
}

func (id *TaskId) GetChecksumFile() string {
	return path.Join(id.GetOutputDirectory(), ".checksum")
}

func (id *TaskId) LoadChecksum() ([]byte, error) {
	checksumString, err := os.ReadFile(id.GetChecksumFile())
	if err != nil {
		return nil, err
	}

	checksum, err := base64.StdEncoding.DecodeString(string(checksumString))
	if err != nil {
		return nil, err
	}

	return checksum, nil
}

type Task struct {
	Id TaskId

	Inputs []string
	Params cue.Value

	checksum []byte
}

func New(backend, id string, params cue.Value, inputs ...string) Task {
	return Task{
		Id: TaskId{
			backend: backend,
			id:      id,
		},
		Inputs: inputs,
		Params: params,
	}
}

func (t *Task) Backend() string {
	return t.Id.backend
}

func (t *Task) GetOutputDirectory() string {
	return t.Id.GetOutputDirectory()
}

func (t *Task) GenerateChecksum() ([]byte, error) {
	// Check the cached checksum
	if t.checksum != nil {
		return t.checksum, nil
	}

	hasher := sha256.New()

	// Hash the backend name
	hasher.Write([]byte(t.Id.backend))

	// Hash the input files
	for _, file := range t.Inputs {
		fileBytes, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to hash input file %s: %w", file, err)
		}
		hasher.Write(fileBytes)
	}

	// Hash the parameters
	paramsJson, err := t.Params.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task params: %w", err)
	}
	hasher.Write(paramsJson)

	// Cache the checksum for next access
	t.checksum = hasher.Sum(nil)
	return t.checksum, nil
}

func (t *Task) SaveChecksum() error {
	checksum, err := t.GenerateChecksum()
	if err != nil {
		return err
	}

	checksumString := base64.StdEncoding.EncodeToString(checksum)

	return os.WriteFile(
		t.Id.GetChecksumFile(),
		[]byte(checksumString),
		os.ModePerm,
	)
}

func (t *Task) CheckChecksum() bool {
	savedChecksum, _ := t.Id.LoadChecksum()
	if savedChecksum != nil {
		newChecksum, _ := t.GenerateChecksum()

		if reflect.DeepEqual(savedChecksum, newChecksum) {
			return true
		}
	}

	return false
}
