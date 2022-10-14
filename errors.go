package task

import (
	"errors"
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/interp"
)

var (
	// ErrTaskfileAlreadyExists is returned on creating a Taskfile if one already exists
	ErrTaskfileAlreadyExists = errors.New("task: A Taskfile already exists")
)

type taskNotFoundError struct {
	taskName   string
	didYouMean string
}

func (err *taskNotFoundError) Error() string {
	if err.didYouMean != "" {
		return fmt.Sprintf(
			`task: Task %q does not exist. Did you mean %q?`,
			err.taskName,
			err.didYouMean,
		)
	}

	return fmt.Sprintf(`task: Task %q does not exist`, err.taskName)
}

type multipleTasksWithAliasError struct {
	aliasName string
	taskNames []string
}

func (err *multipleTasksWithAliasError) Error() string {
	return fmt.Sprintf(`task: Multiple tasks (%s) with alias %q found`, strings.Join(err.taskNames, ", "), err.aliasName)
}

type taskInternalError struct {
	taskName string
}

func (err *taskInternalError) Error() string {
	return fmt.Sprintf(`task: Task "%s" is internal`, err.taskName)
}

type TaskRunError struct {
	taskName string
	err      error
}

func (err *TaskRunError) Error() string {
	return fmt.Sprintf(`task: Failed to run task %q: %v`, err.taskName, err.err)
}

func (err *TaskRunError) ExitCode() int {
	if c, ok := interp.IsExitStatus(err.err); ok {
		return int(c)
	}

	return 1
}

// MaximumTaskCallExceededError is returned when a task is called too
// many times. In this case you probably have a cyclic dependendy or
// infinite loop
type MaximumTaskCallExceededError struct {
	task string
}

func (e *MaximumTaskCallExceededError) Error() string {
	return fmt.Sprintf(
		`task: maximum task call exceeded (%d) for task %q: probably an cyclic dep or infinite loop`,
		MaximumTaskCall,
		e.task,
	)
}
