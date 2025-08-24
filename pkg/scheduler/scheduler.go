// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package scheduler // import "go.bonk.build/pkg/scheduler"

import (
	"fmt"
	"log/slog"

	gotaskflow "github.com/noneback/go-taskflow"

	"go.bonk.build/pkg/task"
)

type TaskSender interface {
	SendTask(tsk task.Task) error
}

type Scheduler struct {
	backendManager TaskSender
	executor       gotaskflow.Executor
	tasks          map[string]*gotaskflow.Task
	rootFlow       *gotaskflow.TaskFlow
}

func NewScheduler(backendManager TaskSender, concurrency uint) *Scheduler {
	return &Scheduler{
		backendManager: backendManager,
		executor:       gotaskflow.NewExecutor(concurrency),
		tasks:          make(map[string]*gotaskflow.Task),
		rootFlow:       gotaskflow.NewTaskFlow("bonk"),
	}
}

func (s *Scheduler) AddTask(tsk task.Task, deps ...string) error {
	taskName := tsk.ID.String()
	newTask := s.rootFlow.NewTask(taskName, func() {
		err := s.backendManager.SendTask(tsk)
		if err != nil {
			slog.Error("error executing task", "task", taskName, "error", err)
		}
	})

	for _, dep := range deps {
		depTask, ok := s.tasks[dep]
		if !ok {
			return fmt.Errorf("could not find dependency task %s", dep)
		}

		newTask.Succeed(depTask)
	}

	s.tasks[taskName] = newTask

	return nil
}

func (s *Scheduler) Run() {
	s.executor.Run(s.rootFlow).Wait()
}
