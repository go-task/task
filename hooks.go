package task

import (
	"context"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile"
)

func (e *Executor) hookBeforeAll(ctx context.Context, t *taskfile.Task) {
	if t.Hooks != nil && len(t.Hooks.BeforeAll) != 0 {
		var err error
		for _, cmd := range t.Hooks.BeforeAll {
			if err = e.runCommand(ctx, t, cmd); err != nil {
				e.Logger.VerboseErrf(logger.Red, "task: error executing command in before_all hook: %v", err)
			}
		}
	}
}

func (e *Executor) hookAfterAll(ctx context.Context, t *taskfile.Task) {
	if t.Hooks != nil && len(t.Hooks.AfterAll) != 0 {
		var err error
		for _, cmd := range t.Hooks.AfterAll {
			if err = e.runCommand(ctx, t, cmd); err != nil {
				e.Logger.VerboseErrf(logger.Red, "task: error executing command in after_all hook: %v", err)
			}
		}
	}
}

func (e *Executor) hookSuccess(ctx context.Context, t *taskfile.Task) {
	if t.Hooks != nil && len(t.Hooks.OnSuccess) != 0 {
		var err error
		for _, cmd := range t.Hooks.OnSuccess {
			if err = e.runCommand(ctx, t, cmd); err != nil {
				e.Logger.VerboseErrf(logger.Red, "task: error executing command in on_success hook: %v", err)
			}
		}
	}
}

func (e *Executor) hookFailure(ctx context.Context, t *taskfile.Task) {
	if t.Hooks != nil && len(t.Hooks.OnFailure) != 0 {
		var err error
		for _, cmd := range t.Hooks.OnFailure {
			if err = e.runCommand(ctx, t, cmd); err != nil {
				e.Logger.VerboseErrf(logger.Red, "task: error executing command in on_failure hook: %v", err)
			}
		}
	}
}

func (e *Executor) hookSkipped(ctx context.Context, t *taskfile.Task) {
	if t.Hooks != nil && len(t.Hooks.OnSkipped) != 0 {
		var err error
		for _, cmd := range t.Hooks.OnSkipped {
			if err = e.runCommand(ctx, t, cmd); err != nil {
				e.Logger.VerboseErrf(logger.Red, "task: error executing command in on_skipped hook: %v", err)
			}
		}
	}
}
