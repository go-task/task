package task

import (
	"errors"
	"fmt"

	"github.com/go-task/task/v3/taskfile"
)

var (
	// ErrTaskfileAlreadyExists is returned on creating a Taskfile if one already exists
	ErrTaskfileAlreadyExists = errors.New("task: A Taskfile already exists")
)

type taskNotFoundError struct {
	taskName string
}

func (err *taskNotFoundError) Error() string {
	return fmt.Sprintf(`task: Task "%s" not found`, err.taskName)
}

// RunError is used when a task run runs into an error.
type RunError struct {
	taskName  string

	// Exported for assertion in tests.
	ActualErr error
}

func (err *RunError) Error() string {
	return fmt.Sprintf(`task: Failed to run task "%s": %v`, err.taskName, err.ActualErr)
}

// MaximumTaskCallExceededError is returned when a task is called too
// many times. In this case you probably have a cyclic dependendy or
// infinite loop
type MaximumTaskCallExceededError struct {
	task string
}

func (e *MaximumTaskCallExceededError) Error() string {
	return fmt.Sprintf(
		`task: maximum task call exceeded (%d) for task "%s": probably an cyclic dep or infinite loop`,
		MaximumTaskCall,
		e.task,
	)
}

// InfiniteCallLoopError is used when we detect that a task call is going into an infinite loop.
// Example:
// If a given call is already part of a given call stack, we are in trouble.
// E.g. if from a we call b, then b calls c, and c calls b, (call stack "a,b,c,b")
// then we know we are in trouble, because b was already called and will call c again.
type InfiniteCallLoopError struct {
	// Name of the task that is causing the infinite loop.
	causeTask string

	callStack taskfile.CallStack
}

func (e InfiniteCallLoopError) Error() string {
	return fmt.Sprintf(
		"task %s runs %s, but %s was already run; assuming infinite loop",
		e.causeTask,
		e.callStack[len(e.callStack)-1].Task,
		e.callStack[len(e.callStack)-1].Task,
	)
}

// MaxDepLevelReachedError is used when while analyzing dependencies, we pass the maximum level of depth.
type MaxDepLevelReachedError struct {
	level int
}

func (e MaxDepLevelReachedError) Error() string {
	return fmt.Sprintf("maximum dependency level (%d) exceeded", e.level)
}

// DirectDepCycleError is used when for example A depends on B depends on A.
// Indirect would be when A depends on B depends on C depends on A.
// We use another error in that case.
type DirectDepCycleError struct {
	task1 string
	task2 string
}

func (e DirectDepCycleError) Error() string {
	return fmt.Sprintf("cyclic dependency detected: task %s depends on %s, which depends on %s", e.task1, e.task2, e.task1)
}
