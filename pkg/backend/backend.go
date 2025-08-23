// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package backend // import "go.bonk.build/pkg/backend"

import (
	"context"

	"cuelang.org/go/cue"

	"go.bonk.build/pkg/task"
)

type Backend interface {
	Outputs() []string
	Execute(ctx context.Context, cuectx *cue.Context, tsk task.Task) error
}
