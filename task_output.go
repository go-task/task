package task

import (
	"context"
	"io"

	"github.com/fatih/color"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/output"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

type taskWritersKey struct{}

type taskWriters struct {
	stdout, stderr io.Writer
}

// writersFromCtx returns the task-scoped writers if set, otherwise the
// Executor's own stdout/stderr.
func (e *Executor) writersFromCtx(ctx context.Context) (io.Writer, io.Writer) {
	if tw, ok := ctx.Value(taskWritersKey{}).(*taskWriters); ok && tw != nil {
		return tw.stdout, tw.stderr
	}
	return e.Stdout, e.Stderr
}

// wrapTaskOutput wraps a task's output in a task-scoped block if e.Output
// implements [output.TaskWrapper] and the task is not interactive. Returns
// the (possibly updated) ctx and a closer that flushes the block. The closer
// is always safe to call — it is a no-op when no wrapping took place.
func (e *Executor) wrapTaskOutput(ctx context.Context, t *ast.Task, call *Call) (context.Context, func(error)) {
	noop := func(error) {}
	if t.Interactive {
		return ctx, noop
	}
	tw, ok := e.Output.(output.TaskWrapper)
	if !ok {
		return ctx, noop
	}
	stdOut, stdErr := e.writersFromCtx(ctx)
	vars, err := e.Compiler.FastGetVariables(t, call)
	if err != nil {
		e.Logger.VerboseErrf(logger.Yellow, "task: output setup: %v\n", err)
		return ctx, noop
	}
	wOut, wErr, closer := tw.WrapTask(stdOut, stdErr, &templater.Cache{Vars: vars})
	ctx = context.WithValue(ctx, taskWritersKey{}, &taskWriters{stdout: wOut, stderr: wErr})
	return ctx, func(loopErr error) {
		if err := closer(loopErr); err != nil {
			e.Logger.Errf(logger.Red, "task: output close: %v\n", err)
		}
	}
}

// printCmdAnnouncement prints the "task: [NAME] CMD" line using the
// task-scoped stderr if available, so the announcement ends up inside the
// task's output block.
func (e *Executor) printCmdAnnouncement(ctx context.Context, t *ast.Task, cmdStr string) {
	_, stdErr := e.writersFromCtx(ctx)
	if stdErr == e.Stderr {
		// No task-scoped writer — fall back to the Logger to preserve existing
		// behavior (respects Logger's color config, etc.).
		e.Logger.Errf(logger.Green, "task: [%s] %s\n", t.Name(), cmdStr)
		return
	}
	_, _ = color.New(color.FgGreen).Fprintf(stdErr, "task: [%s] %s\n", t.Name(), cmdStr)
}
