package output

import (
	"context"
	"fmt"
	"io"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

type Output interface {
	WrapWriter(stdOut, stdErr io.Writer, prefix string, cache *templater.Cache) (io.Writer, io.Writer, CloseFunc)
}

type CloseFunc func(err error) error

// TaskInvocation identifies one runtime invocation of a task. IDs are unique
// within an Executor; ParentID is zero for tasks invoked from the CLI.
type TaskInvocation struct {
	ID       uint64
	ParentID uint64
	Name     string
}

// Runner is implemented by output modes that need to own the terminal while
// tasks execute.
type Runner interface {
	Run(context.Context, func(context.Context) error) error
}

// TaskLifecycle is implemented by output modes that display task state in
// addition to command output.
type TaskLifecycle interface {
	TaskStarted(TaskInvocation)
	TaskFinished(id uint64, err error)
}

// TaskOutput is implemented by output modes that keep output for individual
// task invocations. Other output modes continue to use Output.WrapWriter.
type TaskOutput interface {
	WrapWriterForTask(stdOut, stdErr io.Writer, task TaskInvocation, cache *templater.Cache) (io.Writer, io.Writer, CloseFunc)
}

// Run executes fn through the output mode when it needs to manage the whole
// execution lifecycle. Traditional stream-based output modes simply call fn.
func Run(o Output, ctx context.Context, fn func(context.Context) error) error {
	if runner, ok := o.(Runner); ok {
		return runner.Run(ctx, fn)
	}
	return fn(ctx)
}

func TaskStarted(o Output, task TaskInvocation) {
	if lifecycle, ok := o.(TaskLifecycle); ok {
		lifecycle.TaskStarted(task)
	}
}

func TaskFinished(o Output, id uint64, err error) {
	if lifecycle, ok := o.(TaskLifecycle); ok {
		lifecycle.TaskFinished(id, err)
	}
}

// WrapWriter returns task-aware writers when the output mode supports them.
func WrapWriter(o Output, stdOut, stdErr io.Writer, task TaskInvocation, cache *templater.Cache) (io.Writer, io.Writer, CloseFunc) {
	if taskOutput, ok := o.(TaskOutput); ok {
		return taskOutput.WrapWriterForTask(stdOut, stdErr, task, cache)
	}
	return o.WrapWriter(stdOut, stdErr, task.Name, cache)
}

func IsTUI(o Output) bool {
	_, ok := o.(*TUI)
	return ok
}

// Build the Output for the requested ast.Output.
func BuildFor(o *ast.Output, logger *logger.Logger) (Output, error) {
	switch o.Name {
	case "interleaved", "":
		if err := checkOutputGroupUnset(o); err != nil {
			return nil, err
		}
		return Interleaved{}, nil
	case "group":
		return Group{
			Begin:     o.Group.Begin,
			End:       o.Group.End,
			ErrorOnly: o.Group.ErrorOnly,
		}, nil
	case "prefixed":
		if err := checkOutputGroupUnset(o); err != nil {
			return nil, err
		}
		return NewPrefixed(logger), nil
	case "tui":
		if err := checkOutputGroupUnset(o); err != nil {
			return nil, err
		}
		return NewTUI(logger)
	default:
		return nil, fmt.Errorf(`task: output style %q not recognized`, o.Name)
	}
}

func checkOutputGroupUnset(o *ast.Output) error {
	if o.Group.IsSet() {
		return fmt.Errorf("task: output style %q does not support the group begin/end parameter", o.Name)
	}
	return nil
}
