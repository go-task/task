package output

import (
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

// TaskInvocation identifies one runtime call to a task. IDs are unique within
// an Executor, including repeated calls to the same task.
type TaskInvocation struct {
	ID       uint64 // Unique call ID
	ParentID uint64 // ID of the task call that scheduled this call; zero for roots
	RootID   uint64 // ID of the root call requested by the user
	Name     string
}

// TaskLifecycle is implemented by output consumers that also track task state.
type TaskLifecycle interface {
	TaskScheduled(TaskInvocation)
	TaskStarted(TaskInvocation)
	TaskFinished(id uint64, err error)
}

// TaskJoinLifecycle is implemented by output consumers that distinguish calls
// from executions. A joined call waits for the execution owned by ownerID and
// does not produce its own output.
type TaskJoinLifecycle interface {
	TaskJoined(id, ownerID uint64)
}

// TaskOutput is implemented by output consumers that keep output for individual
// task invocations. Other consumers continue to use Output.WrapWriter.
type TaskOutput interface {
	WrapWriterForTask(stdOut, stdErr io.Writer, task TaskInvocation, cache *templater.Cache) (io.Writer, io.Writer, CloseFunc)
}

func TaskScheduled(o Output, task TaskInvocation) {
	if lifecycle, ok := o.(TaskLifecycle); ok {
		lifecycle.TaskScheduled(task)
	}
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

func TaskJoined(o Output, id, ownerID uint64) {
	if lifecycle, ok := o.(TaskJoinLifecycle); ok {
		lifecycle.TaskJoined(id, ownerID)
	}
}

// WrapWriter returns task-aware writers when the output mode supports them.
func WrapWriter(o Output, stdOut, stdErr io.Writer, task TaskInvocation, cache *templater.Cache) (io.Writer, io.Writer, CloseFunc) {
	if taskOutput, ok := o.(TaskOutput); ok {
		return taskOutput.WrapWriterForTask(stdOut, stdErr, task, cache)
	}
	return o.WrapWriter(stdOut, stdErr, task.Name, cache)
}

// TerminalUI marks an Output that owns the terminal while tasks are running.
type TerminalUI interface {
	IsTerminalUI()
}

// IsTerminalUI reports whether an Output owns the terminal while tasks run.
func IsTerminalUI(o Output) bool {
	_, ok := o.(TerminalUI)
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
